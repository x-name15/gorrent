package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestPirateBayScraper_Mock(t *testing.T) {
	mockJSON := `[
		{
			"id": "12345",
			"name": "Test Movie 2021",
			"info_hash": "EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE",
			"leechers": "20",
			"seeders": "300",
			"size": "2048000",
			"added": "1600000000"
		}
	]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJSON))
	}))
	defer ts.Close()

	s := scraper.NewPirateBay()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	results, err := s.Search(context.Background(), "Test Movie")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Test Movie 2021" {
		t.Errorf("Expected 'Test Movie 2021', got '%s'", results[0].Name)
	}
}

func TestPirateBayScraper_SentinelRetry(t *testing.T) {
	sentinelJSON := `[{"id":"0","name":"No results returned","info_hash":"0000000000000000000000000000000000000000"}]`
	realJSON := `[{"id":"999","name":"Metallica 720p","info_hash":"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF","leechers":"1","seeders":"50","size":"10000","added":"1600000000"}]`

	var callsWithoutCat int
	var callsWithCat int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.RawQuery, "cat=0") {
			callsWithCat++
			w.Write([]byte(realJSON))
		} else {
			callsWithoutCat++
			w.Write([]byte(sentinelJSON))
		}
	}))
	defer ts.Close()

	s := scraper.NewPirateBay()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	results, err := s.Search(context.Background(), "metallica")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result via sentinel retry, got %d", len(results))
	}
	if results[0].Name != "Metallica 720p" {
		t.Errorf("Expected 'Metallica 720p', got '%s'", results[0].Name)
	}

	// Verify that the scraper actually made the initial call AND the retry call
	if callsWithoutCat != 1 {
		t.Errorf("expected 1 initial call without cat=0, got %d", callsWithoutCat)
	}
	if callsWithCat != 1 {
		t.Errorf("expected 1 retry call with cat=0, got %d", callsWithCat)
	}
}

package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
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

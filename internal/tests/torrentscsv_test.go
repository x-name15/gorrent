package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestTorrentsCSVScraper_Mock(t *testing.T) {
	mockJSON := `{
		"torrents": [
			{
				"infohash": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"name": "Test CSV Movie",
				"size_bytes": 1024,
				"seeders": 15,
				"leechers": 2
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJSON))
	}))
	defer ts.Close()

	s := scraper.NewTorrentsCSV()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	results, err := s.Search(context.Background(), "Test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Test CSV Movie" {
		t.Errorf("Expected 'Test CSV Movie', got '%s'", results[0].Name)
	}
}

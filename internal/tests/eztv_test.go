package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestEZTVScraper_Mock(t *testing.T) {
	mockJSON := `{
		"torrents": [
			{
				"title": "Test Show S01E01",
				"filename": "Test.Show.S01E01.mkv",
				"hash": "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
				"magnet_url": "magnet:?xt=urn:btih:CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
				"seeds": 100,
				"peers": 5,
				"size_bytes": 1024000,
				"date_released_unix": 1600000000
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJSON))
	}))
	defer ts.Close()

	s := scraper.NewEZTV()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	// EZTV only supports empty queries for generic API (requires IMDB otherwise)
	results, err := s.Search(context.Background(), "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Test Show S01E01" {
		t.Errorf("Expected 'Test Show S01E01', got '%s'", results[0].Name)
	}
	if results[0].Seeders != 100 {
		t.Errorf("Expected 100 seeders, got %d", results[0].Seeders)
	}
}

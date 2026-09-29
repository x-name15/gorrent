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
				"date_released_unix": 1600000000,
				"imdb_id": "12345"
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

	// 1. Empty query (browse latest)
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

	// 2. Query with text should now match via recent feed index and IMDb lookup!
	s2 := scraper.NewEZTV()
	if inj, ok := s2.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}
	queryResults, err := s2.Search(context.Background(), "Test Show")
	if err != nil {
		t.Fatalf("Expected no error on query search, got %v", err)
	}
	if len(queryResults) == 0 {
		t.Fatalf("Expected results for 'Test Show', got 0")
	}
	if queryResults[0].Name != "Test Show S01E01" {
		t.Errorf("Expected 'Test Show S01E01', got '%s'", queryResults[0].Name)
	}

	// 3. Query with non-matching text must return 0 results (verifies we don't blindly return rows)
	noResults, err := s2.Search(context.Background(), "UnrelatedMovie2026")
	if err != nil {
		t.Fatalf("Expected no error on non-matching query, got %v", err)
	}
	if len(noResults) != 0 {
		t.Fatalf("Expected 0 results for non-matching query, got %d", len(noResults))
	}
}

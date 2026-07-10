package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestBitTorrentedScraper_Mock(t *testing.T) {
	// Create a mock HTTP server that returns a valid JSON response
	mockResponse := `{
		"results": [
			{
				"torrent_infohash": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				"torrent_name": "Test.Movie.1080p",
				"torrent_total_size": 1048576,
				"torrent_seeders": 42,
				"torrent_leechers": 5
			},
			{
				"torrent_infohash": "invalid-hash",
				"torrent_name": "Should.Be.Dropped",
				"torrent_total_size": 0,
				"torrent_seeders": 0,
				"torrent_leechers": 0
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search/torrents" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	s := scraper.NewBitTorrented()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}
	
	// Execute search
	results, err := s.Search(context.Background(), "Test Movie")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// We expect exactly 1 valid result (the second one has an invalid infohash)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Test.Movie.1080p" {
		t.Errorf("Expected Name 'Test.Movie.1080p', got '%s'", results[0].Name)
	}

	if results[0].Seeders != 42 {
		t.Errorf("Expected 42 seeders, got %d", results[0].Seeders)
	}
}

package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestYTSScraper_Mock(t *testing.T) {
	mockJSON := `{
		"data": {
			"movies": [
				{
					"title_long": "Test YTS Movie",
					"torrents": [
						{
							"hash": "cccccccccccccccccccccccccccccccccccccccc",
							"quality": "1080p",
							"size_bytes": 1024,
							"seeds": 20,
							"peers": 3
						}
					]
				}
			]
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJSON))
	}))
	defer ts.Close()

	s := scraper.NewYTS()
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

	if results[0].Name != "Test YTS Movie [1080p]" {
		t.Errorf("Expected 'Test YTS Movie [1080p]', got '%s'", results[0].Name)
	}
}

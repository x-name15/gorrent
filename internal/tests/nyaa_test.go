package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestNyaaScraper_Mock(t *testing.T) {
	mockXML := `
	<rss version="2.0">
		<channel>
			<item>
				<title>Test Anime Episode 1</title>
				<link>https://nyaa.si/view/123</link>
				<seeders>500</seeders>
				<leechers>10</leechers>
				<size>500 MiB</size>
				<infoHash>DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD</infoHash>
			</item>
		</channel>
	</rss>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockXML))
	}))
	defer ts.Close()

	s := scraper.NewNyaa()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	results, err := s.Search(context.Background(), "Test Anime")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Test Anime Episode 1" {
		t.Errorf("Expected 'Test Anime Episode 1', got '%s'", results[0].Name)
	}
	if results[0].Seeders != 500 {
		t.Errorf("Expected 500 seeders, got %d", results[0].Seeders)
	}
}

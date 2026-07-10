package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func TestFitGirlScraper_Mock(t *testing.T) {
	mockXML := `
	<rss>
		<channel>
			<item>
				<title><![CDATA[Test Game [FitGirl Repack]]]></title>
				<content:encoded><![CDATA[
					<a href="magnet:?xt=urn:btih:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB">Magnet</a>
				]]></content:encoded>
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

	s := scraper.NewFitGirl()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	results, err := s.Search(context.Background(), "Test Game")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Test Game [FitGirl Repack]" {
		t.Errorf("Expected 'Test Game [FitGirl Repack]', got '%s'", results[0].Name)
	}
	if results[0].InfoHash != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Errorf("Expected infohash 'bbbb...', got '%s'", results[0].InfoHash)
	}
}

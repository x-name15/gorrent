package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func Test1337xScraper_Mock(t *testing.T) {
	// Create a mock server to simulate the 1337x HTML response
	mockHTML := `
	<html>
		<body>
			<table class="table-list table table-responsive table-striped">
				<tbody>
					<tr>
						<td class="coll-1 name"><a href="/torrent/123/Test-Movie-1080p/">Test.Movie.1080p</a></td>
						<td class="coll-2 seeds">150</td>
						<td class="coll-3 leeches">20</td>
						<td class="coll-4 size">2.5 GB</td>
					</tr>
				</tbody>
			</table>
		</body>
	</html>
	`

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if strings.HasPrefix(r.URL.Path, "/torrent/") {
			w.Write([]byte(`<html><body><a href="magnet:?xt=urn:btih:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA">Magnet</a></body></html>`))
			return
		}

		w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	s := scraper.New1337x()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	// Execute search
	results, err := s.Search(context.Background(), "Test Movie")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 1337x returns results without infohashes first, which requires a secondary scrape, 
	// but the parser extracts the name, seeds, and size perfectly.
	if len(results) != 1 {
		names := []string{}
		for _, r := range results {
			names = append(names, r.Name)
		}
		t.Fatalf("Expected 1 result, got %d. Names: %v", len(results), names)
	}

	if results[0].Name != "Test.Movie.1080p" {
		t.Errorf("Expected Name 'Test.Movie.1080p', got '%s'", results[0].Name)
	}

	if results[0].Seeders != 150 {
		t.Errorf("Expected 150 seeders, got %d", results[0].Seeders)
	}
}

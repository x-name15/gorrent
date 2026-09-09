package tests

import (
	"context"
	"encoding/base32"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
)

func Test1337xScraper_Mock(t *testing.T) {
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

	results, err := s.Search(context.Background(), "Test Movie")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

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

func Test1337xScraper_Base32Normalization(t *testing.T) {
	mockHTML := `
	<html>
		<body>
			<table class="table-list table table-responsive table-striped">
				<tbody>
					<tr>
						<td class="coll-1 name"><a href="/torrent/456/Base32-Movie/">Base32.Movie.720p</a></td>
						<td class="coll-2 seeds">80</td>
						<td class="coll-3 leeches">10</td>
						<td class="coll-4 size">1.5 GB</td>
					</tr>
				</tbody>
			</table>
		</body>
	</html>
	`

	// Valid 32-char RFC 4648 Base32 hash (20 bytes)
	rawBytes := []byte("12345678901234567890")
	base32Hash := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawBytes)
	expectedHex := hex.EncodeToString(rawBytes)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if strings.HasPrefix(r.URL.Path, "/torrent/") {
			w.Write([]byte(`<html><body><a href="magnet:?xt=urn:btih:` + base32Hash + `&dn=Base32">Magnet</a></body></html>`))
			return
		}

		w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	s := scraper.New1337x()
	if inj, ok := s.(clientInjector); ok {
		inj.InjectClient(newMockClient(ts))
	}

	results, err := s.Search(context.Background(), "Base32 Movie")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Must be accurately decoded and normalized to exact 40-character hex string
	if results[0].InfoHash != expectedHex {
		t.Errorf("expected exact normalized hex infohash %s, got %s", expectedHex, results[0].InfoHash)
	}
}

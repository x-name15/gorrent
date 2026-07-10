package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/scraper"
	"golang.org/x/text/encoding/charmap"
)

func TestRuTrackerScraper_Mock(t *testing.T) {
	mockHTML := `
	<table class="tor-tbl">
	<tr class="tCenter hl-tr">
		<td class="tL"></td>
		<td data-ts_text="12345">
			<a class="tLink" href="viewtopic.php?t=12345">Test RuTracker Movie</a>
		</td>
		<td class="tor-size" data-ts_text="1024">1 KB</td>
		<td class="seedmed" data-ts_text="50">50</td>
		<td class="leechmed" data-ts_text="5">5</td>
	</tr>
	</table>
	`

	mockThread := `
	<html>
		<body>
			<a href="magnet:?xt=urn:btih:EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE">Magnet</a>
		</body>
	</html>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=windows-1251")
		w.WriteHeader(http.StatusOK)

		encoder := charmap.Windows1251.NewEncoder()
		var outStr string
		var err error

		if strings.Contains(r.URL.Path, "viewtopic.php") || strings.Contains(r.URL.RawQuery, "t=12345") {
			outStr, err = encoder.String(mockThread)
		} else {
			outStr, err = encoder.String(mockHTML)
		}

		if err == nil {
			w.Write([]byte(outStr))
		}
	}))
	defer ts.Close()

	s := scraper.NewRuTracker("dummy-cookie")
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

	if results[0].Name != "Test RuTracker Movie" {
		t.Errorf("Expected 'Test RuTracker Movie', got '%s'", results[0].Name)
	}
	if results[0].Seeders != 50 {
		t.Errorf("Expected 50 seeders, got %d", results[0].Seeders)
	}
}

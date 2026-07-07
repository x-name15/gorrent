package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var x1337Hosts = []string{"1337x.to", "1337x.st", "x1337x.ws", "1337xx.to"}

type x1337 struct {
	client *http.Client
}

func New1337x() search.Source {
	return &x1337{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *x1337) ID() string {
	return "x1337"
}

func (s *x1337) Name() string {
	return "1337x"
}

func (s *x1337) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil // 1337x requires a search term
	}

	path := fmt.Sprintf("/search/%s/1/", url.QueryEscape(q))

	var html string
	var base string
	var lastErr error

	for _, host := range x1337Hosts {
		candidate := fmt.Sprintf("https://%s", host)
		u := fmt.Sprintf("%s%s", candidate, path)
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)") // 1337x blocks default agents

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			html = string(b)
			base = candidate
			resp.Body.Close()
			break
		}
		resp.Body.Close()
		lastErr = fmt.Errorf("1337x returned %d", resp.StatusCode)
	}

	if base == "" {
		return nil, fmt.Errorf("all 1337x hosts failed, last error: %v", lastErr)
	}

	return parse1337xRows(ctx, s.client, html, base)
}

func parse1337xRows(ctx context.Context, client *http.Client, html string, base string) ([]search.TorrentResult, error) {
	// Simple string manipulation to extract rows
	startIdx := strings.Index(html, "table-list")
	if startIdx < 0 {
		return nil, nil
	}
	html = html[startIdx:]

	rows := strings.Split(html, "<tr")
	var results []search.TorrentResult

	reLink := regexp.MustCompile(`href="(/torrent/[^"]+)"[^>]*>([^<]+)</a>`)
	reSeeders := regexp.MustCompile(`class="coll-2 seeds[^"]*">\s*(\d+)`)
	reLeechers := regexp.MustCompile(`class="coll-3 leeches[^"]*">\s*(\d+)`)
	reSize := regexp.MustCompile(`class="coll-4 size[^"]*">\s*([\d.]+\s*[A-Za-z]+)`)

	limit := 10
	count := 0

	for _, tr := range rows {
		if count >= limit {
			break
		}
		linkMatch := reLink.FindStringSubmatch(tr)
		if len(linkMatch) < 3 {
			continue
		}

		path := linkMatch[1]
		name := strings.TrimSpace(linkMatch[2])

		seedersStr := "0"
		if m := reSeeders.FindStringSubmatch(tr); len(m) > 1 {
			seedersStr = m[1]
		}
		leechersStr := "0"
		if m := reLeechers.FindStringSubmatch(tr); len(m) > 1 {
			leechersStr = m[1]
		}

		sizeStr := ""
		if m := reSize.FindStringSubmatch(tr); len(m) > 1 {
			sizeStr = m[1]
		}

		seeders, _ := strconv.Atoi(seedersStr)
		leechers, _ := strconv.Atoi(leechersStr)

		// Fetch magnet
		magnet := fetch1337xMagnet(ctx, client, base, path)
		if magnet == "" {
			continue
		}

		// Extract infohash from magnet
		infoHash := ""
		if m := regexp.MustCompile(`urn:btih:([a-zA-Z0-9]+)`).FindStringSubmatch(magnet); len(m) > 1 {
			infoHash = strings.ToLower(m[1])
		}

		results = append(results, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      name,
			SizeBytes: parseSizeToBytes(sizeStr),
			Seeders:   seeders,
			Leechers:  leechers,
			Source:    "1337x",
			Magnet:    magnet,
		})
		count++
	}

	return results, nil
}

func fetch1337xMagnet(ctx context.Context, client *http.Client, base, path string) string {
	u := fmt.Sprintf("%s%s", base, path)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	html := string(b)

	m := regexp.MustCompile(`magnet:\?xt=urn:btih:[^"'<>\s]+`).FindStringSubmatch(html)
	if len(m) > 0 {
		return m[0]
	}
	return ""
}

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

	"golang.org/x/text/encoding/charmap"
)

var rutrackerHosts = []string{"rutracker.org", "rutracker.net", "rutracker.nl"}

type rutracker struct {
	client *http.Client
	cookie string
}

func NewRuTracker(cookie string) search.Source {
	return &rutracker{
		client: &http.Client{Timeout: 15 * time.Second},
		cookie: cookie,
	}
}

func (s *rutracker) ID() string {
	return "rutracker"
}

func (s *rutracker) Name() string {
	return "RuTracker"
}

func (s *rutracker) decodeCP1251(body io.Reader) (string, error) {
	decoder := charmap.Windows1251.NewDecoder()
	b, err := io.ReadAll(decoder.Reader(body))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *rutracker) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	if s.cookie == "" {
		return nil, fmt.Errorf("rutracker requires authentication cookie")
	}

	q := strings.TrimSpace(query)
	path := "/forum/tracker.php?nm="
	if q != "" {
		// RuTracker expects CP1251 encoding for exact Cyrillic matching.
		// For now, UTF-8 query escaping is used as a fallback for standard ASCII searches.
		path = fmt.Sprintf("/forum/tracker.php?nm=%s", url.QueryEscape(q))
	}

	var html string
	var base string
	var lastErr error

	for _, host := range rutrackerHosts {
		candidate := fmt.Sprintf("https://%s", host)
		u := fmt.Sprintf("%s%s", candidate, path)
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "gorrent/1.0")
		req.Header.Set("Cookie", s.cookie)

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		decoded, err := s.decodeCP1251(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if strings.Contains(decoded, `id="login-form"`) || strings.Contains(decoded, `name="login_username"`) {
			if !strings.Contains(decoded, "tor-tbl") {
				return nil, fmt.Errorf("rutracker session expired or invalid cookie")
			}
		}

		html = decoded
		base = candidate
		break
	}

	if base == "" {
		return nil, fmt.Errorf("all RuTracker hosts failed, last error: %v", lastErr)
	}

	return s.parseRows(ctx, html, base)
}

func (s *rutracker) parseRows(ctx context.Context, html string, base string) ([]search.TorrentResult, error) {
	startIdx := strings.Index(html, "tor-tbl")
	if startIdx < 0 {
		return nil, nil
	}
	html = html[startIdx:]

	rows := strings.Split(html, "<tr")
	var results []search.TorrentResult

	reTopic := regexp.MustCompile(`viewtopic\.php\?t=(\d+)"[^>]*>([\s\S]*?)</a>`)
	reSize := regexp.MustCompile(`data-ts_text="(\d+)"`)
	reSeeders := regexp.MustCompile(`class="[^"]*seedmed[^"]*"[^>]*>\s*(\d+)`)
	reLeechers := regexp.MustCompile(`class="[^"]*leechmed[^"]*"[^>]*>\s*(\d+)`)

	limit := 10
	count := 0

	for _, tr := range rows {
		if count >= limit {
			break
		}
		topicMatch := reTopic.FindStringSubmatch(tr)
		if len(topicMatch) < 3 {
			continue
		}

		topicId := topicMatch[1]
		name := stripTags(topicMatch[2])

		seedersStr := "0"
		if m := reSeeders.FindStringSubmatch(tr); len(m) > 1 {
			seedersStr = m[1]
		}
		leechersStr := "0"
		if m := reLeechers.FindStringSubmatch(tr); len(m) > 1 {
			leechersStr = m[1]
		}

		sizeBytes := int64(0)
		if m := reSize.FindStringSubmatch(tr); len(m) > 1 {
			sizeBytes, _ = strconv.ParseInt(m[1], 10, 64)
		}

		seeders, _ := strconv.Atoi(seedersStr)
		leechers, _ := strconv.Atoi(leechersStr)

		magnet := s.fetchMagnet(ctx, base, topicId)
		if magnet == "" {
			continue
		}

		infoHash := ""
		if m := regexp.MustCompile(`urn:btih:([a-zA-Z0-9]+)`).FindStringSubmatch(magnet); len(m) > 1 {
			infoHash = strings.ToLower(m[1])
		}

		results = append(results, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      name,
			SizeBytes: sizeBytes,
			Seeders:   seeders,
			Leechers:  leechers,
			Source:    "rutracker",
			Magnet:    magnet,
		})
		count++
	}

	return results, nil
}

func (s *rutracker) fetchMagnet(ctx context.Context, base, topicId string) string {
	u := fmt.Sprintf("%s/forum/viewtopic.php?t=%s", base, topicId)
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "gorrent/1.0")
	req.Header.Set("Cookie", s.cookie)

	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	decoded, _ := s.decodeCP1251(resp.Body)
	m := regexp.MustCompile(`magnet:\?xt=urn:btih:[^"'<>\s]+`).FindStringSubmatch(decoded)
	if len(m) > 0 {
		return m[0]
	}
	return ""
}

func stripTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	str := re.ReplaceAllString(html, "")
	str = strings.ReplaceAll(str, "&nbsp;", " ")
	return strings.TrimSpace(str)
}

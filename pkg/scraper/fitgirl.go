package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type fitgirl struct {
	client *http.Client
}

func NewFitGirl() search.Source {
	return &fitgirl{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *fitgirl) ID() string {
	return "fitgirl"
}

func (s *fitgirl) Name() string {
	return "FitGirl"
}

func (s *fitgirl) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)

	base := "https://fitgirl-repacks.site/"
	u := base + "feed/"
	if q != "" {
		u = fmt.Sprintf("%s?s=%s&feed=rss2", base, url.QueryEscape(q))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)") // WP blocks

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FitGirl returned %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	xmlStr := string(b)

	items := strings.Split(xmlStr, "<item>")
	if len(items) <= 1 {
		return nil, nil
	}

	var out []search.TorrentResult
	for _, item := range items[1:] {
		magnetMatch := regexp.MustCompile(`href="(magnet:\?xt=urn:btih:[^"]+)"`).FindStringSubmatch(item)
		if len(magnetMatch) == 0 {
			continue
		}
		magnet := magnetMatch[1]

		infoHashMatch := regexp.MustCompile(`urn:btih:([a-zA-Z0-9]+)`).FindStringSubmatch(magnet)
		if len(infoHashMatch) == 0 {
			continue
		}
		infoHash := strings.ToLower(infoHashMatch[1])

		titleMatch := regexp.MustCompile(`<title><!\[CDATA\[(.*?)\]\]></title>`).FindStringSubmatch(item)
		if len(titleMatch) == 0 {
			titleMatch = regexp.MustCompile(`<title>(.*?)</title>`).FindStringSubmatch(item)
		}
		title := "Unknown"
		if len(titleMatch) > 1 {
			title = titleMatch[1]
		}

		out = append(out, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      title,
			SizeBytes: 0,
			Seeders:   0,
			Leechers:  0,
			Source:    "fitgirl",
			Magnet:    magnet,
		})
	}
	return out, nil
}

package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"encoding/xml"
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

type fgRss struct {
	Channel struct {
		Items []struct {
			Title   string `xml:"title"`
			Encoded string `xml:"encoded"`
			PubDate string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
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

	// This assumes the rss namespace defines content:encoded, which xml.NewDecoder sometimes struggles with without namespaces.
	// But it's often close enough if we just string search, or use a custom XML parser.
	// We'll use a hacky string-based approach because WP RSS content:encoded is tricky.

	// Just for simplicity of the port, let's use xml parser and hope content:encoded maps if we ignore namespace.
	var rss fgRss
	xml.NewDecoder(resp.Body).Decode(&rss)

	// Actually, `content:encoded` is not easily mapped via `xml:"encoded"`.
	// We'd better parse magnet from raw text.
	// Wait, since we are returning a slice, I'll re-do the request to get raw bytes for regex.
	return s.SearchRegexHack(ctx, u)
}

func (s *fitgirl) SearchRegexHack(ctx context.Context, u string) ([]search.TorrentResult, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FitGirl returned %d", resp.StatusCode)
	}

	b, _ := io.ReadAll(resp.Body)
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

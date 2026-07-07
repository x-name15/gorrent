package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type nyaa struct {
	client *http.Client
}

type NyaaRss struct {
	Channel struct {
		Items []NyaaItem `xml:"item"`
	} `xml:"channel"`
}

type NyaaItem struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"`
	Guid      string `xml:"guid"`
	PubDate   string `xml:"pubDate"`
	Seeders   string `xml:"seeders"`
	Leechers  string `xml:"leechers"`
	Downloads string `xml:"downloads"`
	InfoHash  string `xml:"infoHash"`
	Size      string `xml:"size"`
}

func NewNyaa() search.Source {
	return &nyaa{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *nyaa) ID() string {
	return "nyaa"
}

func (s *nyaa) Name() string {
	return "Nyaa"
}

func (s *nyaa) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	params := url.Values{}
	params.Set("page", "rss")
	params.Set("c", "0_0")
	params.Set("f", "0")
	if q != "" {
		params.Set("q", q)
	}

	u := fmt.Sprintf("https://nyaa.si/?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gorrent/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nyaa returned %d", resp.StatusCode)
	}

	var rss NyaaRss
	if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
		return nil, err
	}

	var out []search.TorrentResult
	for _, item := range rss.Channel.Items {
		infoHash := strings.ToLower(item.InfoHash)
		if infoHash == "" || item.Title == "" {
			continue
		}

		seeders, _ := strconv.Atoi(item.Seeders)
		leechers, _ := strconv.Atoi(item.Leechers)
		sizeBytes := parseSizeToBytes(item.Size)

		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(item.Title))

		out = append(out, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      item.Title,
			SizeBytes: sizeBytes,
			Seeders:   seeders,
			Leechers:  leechers,
			Source:    "nyaa",
			Magnet:    magnet,
		})
	}

	return out, nil
}

func parseSizeToBytes(sizeStr string) int64 {
	// e.g. "1.5 GiB"
	parts := strings.Fields(sizeStr)
	if len(parts) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToUpper(parts[1])
	switch unit {
	case "KIB", "KB":
		return int64(val * 1024)
	case "MIB", "MB":
		return int64(val * 1024 * 1024)
	case "GIB", "GB":
		return int64(val * 1024 * 1024 * 1024)
	case "TIB", "TB":
		return int64(val * 1024 * 1024 * 1024 * 1024)
	}
	return int64(val)
}

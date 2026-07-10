package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/x-name15/gorrent/pkg/search"
)

var btBase = "https://bittorrented.com"

type btResponse struct {
	Results []btResult `json:"results"`
}

type btResult struct {
	InfoHash  string `json:"torrent_infohash"`
	Name      string `json:"torrent_name"`
	SizeBytes int64  `json:"torrent_total_size"`
	Seeders   *int   `json:"torrent_seeders"`
	Leechers  *int   `json:"torrent_leechers"`
	CreatedAt string `json:"torrent_created_at"`
}

type bittorrented struct {
	client *http.Client
}

func NewBitTorrented() search.Source {
	return &bittorrented{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *bittorrented) InjectClient(c *http.Client) {
	s.client = c
}

func (s *bittorrented) ID() string   { return "bittorrented" }
func (s *bittorrented) Name() string { return "BitTorrented" }

func (s *bittorrented) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	if len(q) < 3 {
		// API requires at least 3 characters
		return nil, nil
	}

	u := fmt.Sprintf("%s/api/search/torrents?q=%s", btBase, url.QueryEscape(q))
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
		return nil, fmt.Errorf("bittorrented returned %d", resp.StatusCode)
	}

	var result btResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 5<<20)).Decode(&result); err != nil {
		return nil, err
	}

	var out []search.TorrentResult
	for _, r := range result.Results {
		if len(r.InfoHash) != 40 {
			continue // skip invalid hashes
		}
		infoHash := strings.ToLower(r.InfoHash)
		name := r.Name
		if name == "" {
			name = infoHash
		}

		seeders := 0
		if r.Seeders != nil {
			seeders = *r.Seeders
		}
		leechers := 0
		if r.Leechers != nil {
			leechers = *r.Leechers
		}

		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(name))

		out = append(out, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      name,
			SizeBytes: r.SizeBytes,
			Seeders:   seeders,
			Leechers:  leechers,
			Source:    "bittorrented",
			Magnet:    magnet,
		})
	}
	return out, nil
}

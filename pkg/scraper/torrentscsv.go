package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type torrentsCsv struct {
	client *http.Client
}

type tcsvTorrent struct {
	Infohash    string `json:"infohash"`
	Name        string `json:"name"`
	SizeBytes   int64  `json:"size_bytes"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	CreatedUnix int64  `json:"created_unix"`
}

type tcsvResponse struct {
	Torrents []tcsvTorrent `json:"torrents"`
}

func NewTorrentsCSV() search.Source {
	return &torrentsCsv{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *torrentsCsv) ID() string {
	return "torrentscsv"
}

func (s *torrentsCsv) Name() string {
	return "Torrents.csv"
}

func (s *torrentsCsv) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil // No browse endpoint
	}

	u := fmt.Sprintf("https://torrents-csv.com/service/search?q=%s&size=100", url.QueryEscape(q))
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
		return nil, fmt.Errorf("Torrents-CSV returned %d", resp.StatusCode)
	}

	var jsonResp tcsvResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, err
	}

	var out []search.TorrentResult
	for _, t := range jsonResp.Torrents {
		infoHash := strings.ToLower(t.Infohash)
		if infoHash == "" {
			continue
		}
		name := t.Name
		if name == "" {
			name = "Unknown"
		}
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(name))

		out = append(out, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      name,
			SizeBytes: t.SizeBytes,
			Seeders:   t.Seeders,
			Leechers:  t.Leechers,
			Source:    "torrentscsv",
			Magnet:    magnet,
			Added:     t.CreatedUnix,
		})
	}
	return out, nil
}

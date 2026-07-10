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

type eztv struct {
	client *http.Client
}

func NewEZTV() search.Source {
	return &eztv{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *eztv) InjectClient(c *http.Client) {
	s.client = c
}

func (s *eztv) ID() string {
	return "eztv"
}

func (s *eztv) Name() string {
	return "EZTV"
}

type eztvResponse struct {
	Torrents []struct {
		Title            string      `json:"title"`
		Filename         string      `json:"filename"`
		Hash             string      `json:"hash"`
		MagnetUrl        string      `json:"magnet_url"`
		Seeds            int         `json:"seeds"`
		Peers            int         `json:"peers"`
		SizeBytes        interface{} `json:"size_bytes"`
		DateReleasedUnix int64       `json:"date_released_unix"`
	} `json:"torrents"`
}

func (s *eztv) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	if q != "" {
		// EZTV API requires an IMDB ID. Generic text search is unsupported.
		return nil, nil
	}

	u := "https://eztvx.to/api/get-torrents?limit=100&page=1"
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
		return nil, fmt.Errorf("EZTV returned %d", resp.StatusCode)
	}

	var jsonResp eztvResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, err
	}

	var out []search.TorrentResult
	for _, t := range jsonResp.Torrents {
		hash := strings.ToLower(t.Hash)
		if hash == "" {
			continue
		}

		name := t.Title
		if name == "" {
			name = t.Filename
		}
		if name == "" {
			name = hash
		}

		magnet := t.MagnetUrl
		if magnet == "" {
			magnet = fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", hash, url.QueryEscape(name))
		}

		sizeBytes := int64(0)
		switch v := t.SizeBytes.(type) {
		case string:
			fmt.Sscanf(v, "%d", &sizeBytes)
		case float64:
			sizeBytes = int64(v)
		}

		out = append(out, search.TorrentResult{
			InfoHash:  hash,
			Name:      name,
			SizeBytes: sizeBytes,
			Seeders:   t.Seeds,
			Leechers:  t.Peers,
			Source:    "eztv",
			Magnet:    magnet,
			Added:     t.DateReleasedUnix,
		})
	}
	return out, nil
}

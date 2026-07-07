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

var ytsHosts = []string{"yts.am", "yts.rs"}

type ytsResponse struct {
	Data struct {
		Movies []struct {
			TitleLong        string `json:"title_long"`
			Title            string `json:"title"`
			DateUploadedUnix int64  `json:"date_uploaded_unix"`
			Torrents         []struct {
				Hash      string `json:"hash"`
				Quality   string `json:"quality"`
				Type      string `json:"type"`
				SizeBytes int64  `json:"size_bytes"`
				Seeds     int    `json:"seeds"`
				Peers     int    `json:"peers"`
			} `json:"torrents"`
		} `json:"movies"`
	} `json:"data"`
}

type yts struct {
	client *http.Client
}

// NewYTS creates a new YTS scraper instance.
func NewYTS() search.Source {
	return &yts{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *yts) ID() string {
	return "yts"
}

func (s *yts) Name() string {
	return "YTS"
}

func (s *yts) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	params := url.Values{}
	params.Set("limit", "50")
	if q != "" {
		params.Set("query_term", q)
	} else {
		params.Set("sort_by", "date_added")
	}

	var lastErr error
	for _, host := range ytsHosts {
		u := fmt.Sprintf("https://%s/api/v2/list_movies.json?%s", host, params.Encode())
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "gorrent/1.0")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("YTS returned %d", resp.StatusCode)
			continue
		}

		var result ytsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body.Close()

		var out []search.TorrentResult
		for _, m := range result.Data.Movies {
			base := m.TitleLong
			if base == "" {
				base = m.Title
			}
			if base == "" {
				base = "Unknown"
			}
			for _, t := range m.Torrents {
				if t.Hash == "" {
					continue
				}
				infoHash := strings.ToLower(t.Hash)
				tag := ""
				if t.Quality != "" {
					tag += t.Quality
				}
				if t.Type != "" {
					if tag != "" {
						tag += " "
					}
					tag += t.Type
				}

				name := base
				if tag != "" {
					name = fmt.Sprintf("%s [%s]", base, tag)
				}

				magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(name))

				out = append(out, search.TorrentResult{
					InfoHash:  infoHash,
					Name:      name,
					SizeBytes: t.SizeBytes,
					Seeders:   t.Seeds,
					Leechers:  t.Peers,
					Source:    "yts",
					Magnet:    magnet,
					Added:     m.DateUploadedUnix,
				})
			}
		}
		return out, nil
	}

	return nil, fmt.Errorf("all YTS hosts failed, last error: %v", lastErr)
}

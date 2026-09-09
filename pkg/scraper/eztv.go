package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/x-name15/gorrent/pkg/search"
)

const (
	eztvBase       = "https://eztvx.to"
	eztvPageLimit  = 100
	eztvIndexPages = 10
	eztvMaxShows   = 2
	eztvMaxResults = 100
	eztvIndexTTL   = 5 * time.Minute
)

type eztvItem struct {
	Title            string      `json:"title"`
	Filename         string      `json:"filename"`
	Hash             string      `json:"hash"`
	MagnetUrl        string      `json:"magnet_url"`
	Seeds            int         `json:"seeds"`
	Peers            int         `json:"peers"`
	SizeBytes        interface{} `json:"size_bytes"`
	DateReleasedUnix int64       `json:"date_released_unix"`
	ImdbID           string      `json:"imdb_id"`
}

type eztvResponse struct {
	Torrents []eztvItem `json:"torrents"`
}

type eztv struct {
	client    *http.Client
	mu        sync.RWMutex
	cachedAt  time.Time
	indexRows []eztvItem
}

func NewEZTV() search.Source {
	return &eztv{
		client: &http.Client{Timeout: 15 * time.Second},
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

func (s *eztv) fetchPage(ctx context.Context, params url.Values) ([]eztvItem, error) {
	if params.Get("limit") == "" {
		params.Set("limit", fmt.Sprintf("%d", eztvPageLimit))
	}
	u := fmt.Sprintf("%s/api/get-torrents?%s", eztvBase, params.Encode())
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
	return jsonResp.Torrents, nil
}

func (s *eztv) getRecentIndex(ctx context.Context) ([]eztvItem, error) {
	s.mu.RLock()
	if len(s.indexRows) > 0 && time.Since(s.cachedAt) < eztvIndexTTL {
		rows := s.indexRows
		s.mu.RUnlock()
		return rows, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check after acquiring write lock
	if len(s.indexRows) > 0 && time.Since(s.cachedAt) < eztvIndexTTL {
		return s.indexRows, nil
	}

	// Page 1 is critical to determine if source is reachable
	p1, err := s.fetchPage(ctx, url.Values{"page": []string{"1"}})
	if err != nil {
		return nil, err
	}

	allRows := p1
	// Fetch deeper pages (2..eztvIndexPages)
	for p := 2; p <= eztvIndexPages; p++ {
		select {
		case <-ctx.Done():
			break
		default:
		}
		rows, err := s.fetchPage(ctx, url.Values{"page": []string{fmt.Sprintf("%d", p)}})
		if err == nil && len(rows) > 0 {
			allRows = append(allRows, rows...)
		} else if err != nil {
			// Stop on failure or exhaustion
			break
		}
	}

	s.indexRows = allRows
	s.cachedAt = time.Now()
	return allRows, nil
}

func (s *eztv) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)

	// If empty query, return latest popular items
	if q == "" {
		rows, err := s.fetchPage(ctx, url.Values{"page": []string{"1"}})
		if err != nil {
			return nil, err
		}
		return toTorrentResults(rows), nil
	}

	tokens := strings.Fields(strings.ToLower(q))
	recent, err := s.getRecentIndex(ctx)
	if err != nil {
		return nil, err
	}

	var hits []eztvItem
	var showIDs []string
	seenShow := make(map[string]bool)

	for _, item := range recent {
		name := strings.ToLower(item.Title + " " + item.Filename)
		matched := true
		for _, token := range tokens {
			if !strings.Contains(name, token) {
				matched = false
				break
			}
		}
		if matched {
			hits = append(hits, item)
			id := strings.TrimSpace(item.ImdbID)
			if id != "" && id != "0" && !seenShow[id] {
				seenShow[id] = true
				showIDs = append(showIDs, id)
				if len(showIDs) >= eztvMaxShows {
					break
				}
			}
		}
	}

	allCandidates := hits
	for _, imdbID := range showIDs {
		select {
		case <-ctx.Done():
			break
		default:
		}
		catalog, err := s.fetchPage(ctx, url.Values{"page": []string{"1"}, "imdb_id": []string{imdbID}})
		if err == nil {
			for _, item := range catalog {
				name := strings.ToLower(item.Title + " " + item.Filename)
				matched := true
				for _, token := range tokens {
					if !strings.Contains(name, token) {
						matched = false
						break
					}
				}
				if matched {
					allCandidates = append(allCandidates, item)
				}
			}
		}
	}

	results := toTorrentResults(allCandidates)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Seeders > results[j].Seeders
	})

	if len(results) > eztvMaxResults {
		results = results[:eztvMaxResults]
	}

	return results, nil
}

func toTorrentResults(items []eztvItem) []search.TorrentResult {
	byHash := make(map[string]bool)
	var out []search.TorrentResult

	for _, t := range items {
		hash := search.NormalizeInfoHash(t.Hash)
		if hash == "" || byHash[hash] {
			continue
		}
		byHash[hash] = true

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
		case int64:
			sizeBytes = v
		case int:
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
	return out
}

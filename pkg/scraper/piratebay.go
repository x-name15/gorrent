package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type piratebay struct {
	client *http.Client
}

type pbItem struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	Added    string `json:"added"`
}

func NewPirateBay() search.Source {
	return &piratebay{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *piratebay) InjectClient(c *http.Client) {
	s.client = c
}

func (s *piratebay) ID() string {
	return "piratebay"
}

func (s *piratebay) Name() string {
	return "The Pirate Bay"
}

func isNoResultsSentinel(items []pbItem) bool {
	return len(items) == 1 && (items[0].Id == "0" || items[0].Name == "No results returned")
}

func (s *piratebay) fetchItems(ctx context.Context, u string) ([]pbItem, error) {
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
		return nil, fmt.Errorf("PirateBay returned %d", resp.StatusCode)
	}

	var items []pbItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *piratebay) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	u := fmt.Sprintf("https://apibay.org/q.php?q=%s", url.QueryEscape(q))
	items, err := s.fetchItems(ctx, u)
	if err != nil {
		return nil, err
	}

	// If apibay returns its cached "no results" sentinel, retry with explicit cat=0
	// to re-roll against an alternate unpoisoned cache key.
	if isNoResultsSentinel(items) {
		retryURL := fmt.Sprintf("https://apibay.org/q.php?q=%s&cat=0", url.QueryEscape(q))
		retryItems, retryErr := s.fetchItems(ctx, retryURL)
		if retryErr == nil && !isNoResultsSentinel(retryItems) {
			items = retryItems
		}
	}

	var out []search.TorrentResult
	for _, item := range items {
		if item.Id == "0" && item.Name == "No results returned" {
			continue
		}

		infoHash := search.NormalizeInfoHash(item.InfoHash)
		if infoHash == "" || item.Name == "" {
			continue
		}

		seeders, _ := strconv.Atoi(item.Seeders)
		leechers, _ := strconv.Atoi(item.Leechers)
		sizeBytes, _ := strconv.ParseInt(item.Size, 10, 64)
		added, _ := strconv.ParseInt(item.Added, 10, 64)

		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infoHash, url.QueryEscape(item.Name))

		out = append(out, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      item.Name,
			SizeBytes: sizeBytes,
			Seeders:   seeders,
			Leechers:  leechers,
			Source:    "piratebay",
			Magnet:    magnet,
			Added:     added,
		})
	}

	return out, nil
}

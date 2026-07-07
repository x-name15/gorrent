package search

import (
	"context"
)

// TorrentResult represents a single torrent found by a scraper.
type TorrentResult struct {
	InfoHash  string `json:"info_hash"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Source    string `json:"source"`
	Magnet    string `json:"magnet"`
	Added     int64  `json:"added,omitempty"`
	Score     int    `json:"score"` // Computed score based on filters
}

// Source defines the interface that all torrent scrapers must implement.
type Source interface {
	ID() string
	Name() string
	Search(ctx context.Context, query string) ([]TorrentResult, error)
}

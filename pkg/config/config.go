package config

import (
	"encoding/json"
	"os"
)

// Config represents the application configuration.
type Config struct {
	Daemon  DaemonConfig  `json:"daemon"`
	Scraper ScraperConfig `json:"scraper"`
	Torrent TorrentConfig `json:"torrent"`
}

// DaemonConfig holds settings for the background service.
type DaemonConfig struct {
	Port   int    `json:"port"`
	APIKey string `json:"api_key"`
}

// ScraperConfig holds settings for the search engines.
type ScraperConfig struct {
	Sources         []string          `json:"sources"`
	Filters         map[string]string `json:"filters"` // e.g. "language": "spanish", "quality": "1080p"
	DNS             string            `json:"dns"`     // e.g. "cloudflare", "google", "8.8.8.8"
	RutrackerCookie string            `json:"rutracker_cookie"`
}

// TorrentConfig holds settings for downloads.
type TorrentConfig struct {
	DownloadDir  string            `json:"download_dir"`
	AutoExport   bool              `json:"auto_export_torrent"`
	Trackers     []string          `json:"trackers"`
	CategoryDirs map[string]string `json:"category_dirs"`
}

// Load reads and parses a JSON config file.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Default returns a standard configuration if none is provided.
func Default() *Config {
	return &Config{
		Daemon: DaemonConfig{
			Port: 7800,
		},
		Scraper: ScraperConfig{
			Sources: []string{"yts", "1337x", "nyaa", "piratebay"},
			Filters: map[string]string{
				"language": "", // Default no language filter
			},
		},
		Torrent: TorrentConfig{
			DownloadDir: "./downloads",
		},
	}
}

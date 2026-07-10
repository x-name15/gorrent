package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Daemon  DaemonConfig  `json:"daemon" yaml:"daemon"`
	Scraper ScraperConfig `json:"scraper" yaml:"scraper"`
	Torrent TorrentConfig `json:"torrent" yaml:"torrent"`
	RSS     RSSConfig     `json:"rss" yaml:"rss"`
}

// DaemonConfig holds settings for the background service.
type DaemonConfig struct {
	Port    int    `json:"port" yaml:"port"`
	APIKey  string `json:"api_key" yaml:"api_key"`
	DataDir string `json:"data_dir" yaml:"data_dir"` // e.g. "./data" for internal state like rss_history.json
}

// ScraperConfig holds settings for the search engines.
type ScraperConfig struct {
	Sources         []string          `json:"sources" yaml:"sources"`
	Filters         map[string]string `json:"filters" yaml:"filters"` // e.g. "language": "spanish", "quality": "1080p"
	DNS             string            `json:"dns" yaml:"dns"`         // e.g. "cloudflare", "google", "8.8.8.8"
	RutrackerCookie string            `json:"rutracker_cookie" yaml:"rutracker_cookie"`
}

// TorrentConfig holds settings for downloads.
type TorrentConfig struct {
	DownloadDir       string            `json:"download_dir" yaml:"download_dir"`
	AutoExport        bool              `json:"auto_export_torrent" yaml:"auto_export_torrent"`
	Trackers          []string          `json:"trackers" yaml:"trackers"`
	CategoryDirs      map[string]string `json:"category_dirs" yaml:"category_dirs"`
	MaxDownloadRate   int               `json:"max_download_rate" yaml:"max_download_rate"`       // in KB/s
	MaxUploadRate     int               `json:"max_upload_rate" yaml:"max_upload_rate"`           // in KB/s
	AutoCleanup       bool              `json:"auto_cleanup" yaml:"auto_cleanup"`                 // Optional, false by default
	SeedRatio         float64           `json:"seed_ratio" yaml:"seed_ratio"`                     // Target ratio to stop seeding
	MaxSeedDays       int               `json:"max_seed_days" yaml:"max_seed_days"`               // Days to seed before stopping
	HardlinkDir       string            `json:"hardlink_dir" yaml:"hardlink_dir"`                 // Optional, e.g. "/media"
	PostScript        string            `json:"post_script" yaml:"post_script"`                   // Optional, bash script to run on completion
	WatchDir          string            `json:"watch_dir" yaml:"watch_dir"`                       // Optional, empty = disabled
	DeleteFilesOnStop bool              `json:"delete_files_on_stop" yaml:"delete_files_on_stop"` // Optional, false = safe default
}

// RSSFeed holds the configuration for a single RSS feed.
type RSSFeed struct {
	URL      string   `json:"url" yaml:"url"`
	Category string   `json:"category" yaml:"category"`
	Regex    []string `json:"regex" yaml:"regex"` // Patterns to match (e.g. "[SubsPlease] Arcane (1080p)")
}

// RSSConfig holds the configuration for the RSS auto-downloader.
type RSSConfig struct {
	IntervalMin int       `json:"interval_min" yaml:"interval_min"`
	Feeds       []RSSFeed `json:"feeds" yaml:"feeds"`
}

// Load reads and parses a YAML config file.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Default returns a standard configuration if none is provided.
func Default() *Config {
	return &Config{
		Daemon: DaemonConfig{
			Port:    7800,
			DataDir: "./data",
		},
		Scraper: ScraperConfig{
			Sources: []string{"yts", "1337x", "nyaa", "piratebay", "bittorrented"},
			Filters: map[string]string{
				"language": "", // Default no language filter
			},
		},
		Torrent: TorrentConfig{
			DownloadDir: "./downloads",
		},
	}
}

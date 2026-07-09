package torrent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/x-name15/gorrent/pkg/config"
	"golang.org/x/time/rate"
)

type Client struct {
	tc      *torrent.Client
	cfg     *config.TorrentConfig
	dataDir string
}

// NewClient initializes the anacrolix/torrent client.
func NewClient(cfg *config.TorrentConfig, dataDir string) (*Client, error) {
	tcConfig := torrent.NewDefaultClientConfig()
	tcConfig.DataDir = cfg.DownloadDir
	tcConfig.NoUpload = false

	if cfg.MaxDownloadRate > 0 {
		tcConfig.DownloadRateLimiter = rate.NewLimiter(rate.Limit(cfg.MaxDownloadRate*1024), 16*1024)
	}
	if cfg.MaxUploadRate > 0 {
		tcConfig.UploadRateLimiter = rate.NewLimiter(rate.Limit(cfg.MaxUploadRate*1024), 16*1024)
	}

	tc, err := torrent.NewClient(tcConfig)
	if err != nil {
		return nil, err
	}

	client := &Client{
		tc:      tc,
		cfg:     cfg,
		dataDir: dataDir,
	}

	if cfg.AutoCleanup {
		go client.startGC()
	}
	go client.startPostProcessor()

	return client, nil
}

// AddMagnet adds a torrent via magnet link and starts downloading.
func (c *Client) AddMagnet(magnet string, category string) (*torrent.Torrent, error) {
	spec, err := torrent.TorrentSpecFromMagnetUri(magnet)
	if err != nil {
		return nil, err
	}

	targetPath := c.cfg.DownloadDir
	if category != "" {
		if mappedPath, ok := c.cfg.CategoryDirs[category]; ok {
			targetPath = mappedPath
		} else {
			targetPath = filepath.Join(c.cfg.DownloadDir, category)
		}
	}

	spec.Storage = storage.NewFile(targetPath)

	t, _, err := c.tc.AddTorrentSpec(spec)
	if err != nil {
		return nil, err
	}

	// Wait for info to download with a 30-second timeout
	select {
	case <-t.GotInfo():
		// Info downloaded, we can proceed
		info := t.Info()
		if info != nil && c.cfg.AutoExport {
			// Auto-export .torrent file
			filename := filepath.Join(c.cfg.DownloadDir, info.Name+".torrent")
			if f, err := os.Create(filename); err == nil {
				mi := t.Metainfo()
				mi.Write(f)
				f.Close()
				log.Printf("Exported .torrent file: %s", filename)
			} else {
				log.Printf("Failed to export .torrent file: %v", err)
			}
		}
	case <-time.After(30 * time.Second):
		t.Drop() // Clean up resources
		return nil, fmt.Errorf("timeout waiting for torrent metadata (0 seeders or dead torrent)")
	}

	// Download all files
	t.DownloadAll()

	return t, nil
}

// StopTorrent drops an active torrent from the client.
func (c *Client) StopTorrent(hash string) error {
	for _, t := range c.tc.Torrents() {
		if t.InfoHash().HexString() == hash {
			t.Drop()
			return nil
		}
	}
	return fmt.Errorf("torrent not found")
}

// ConnStats returns connection statistics for the torrent client.
func (c *Client) ConnStats() torrent.ConnStats {
	return c.tc.ConnStats()
}

// Status returns info about all current torrents
func (c *Client) Status() []map[string]interface{} {
	var statuses []map[string]interface{}
	for _, t := range c.tc.Torrents() {
		info := t.Info()
		name := "Downloading metadata..."
		if info != nil {
			name = info.Name
		}

		stats := t.Stats()
		statuses = append(statuses, map[string]interface{}{
			"hash":       t.InfoHash().HexString(),
			"name":       name,
			"downloaded": stats.BytesReadData.Int64(),
			"length":     t.Length(),
			"peers":      stats.ActivePeers,
		})
	}
	return statuses
}

// Close shuts down the torrent client.
func (c *Client) Close() {
	c.tc.Close()
	log.Println("Torrent client closed")
}

// startGC periodically checks torrents for cleanup conditions.
func (c *Client) startGC() {
	log.Printf("Garbage Collector enabled (Ratio: %.2f, Max Days: %d)", c.cfg.SeedRatio, c.cfg.MaxSeedDays)
	for {
		time.Sleep(10 * time.Minute)
		for _, t := range c.tc.Torrents() {
			if t.Info() == nil {
				continue // Skip if metadata is not yet downloaded
			}
			length := t.Length()
			if length == 0 || t.BytesCompleted() < length {
				continue // Still downloading or empty
			}

			stats := t.Stats()
			ratio := float64(stats.BytesWrittenData.Int64()) / float64(length)
			drop := false

			// Check Ratio
			if c.cfg.SeedRatio > 0 && ratio >= c.cfg.SeedRatio {
				log.Printf("GC: Dropping %s (Ratio %.2f reached)", t.Name(), ratio)
				drop = true
			}

			// Check Days Completed (using file modification time as heuristic)
			if !drop && c.cfg.MaxSeedDays > 0 {
				path := filepath.Join(c.cfg.DownloadDir, t.Name())
				if stat, err := os.Stat(path); err == nil {
					daysOld := time.Since(stat.ModTime()).Hours() / 24.0
					if daysOld >= float64(c.cfg.MaxSeedDays) {
						log.Printf("GC: Dropping %s (Seeded for %.1f days)", t.Name(), daysOld)
						drop = true
					}
				}
			}

			if drop {
				t.Drop()
			}
		}
	}
}

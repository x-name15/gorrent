package torrent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/logger"
	"golang.org/x/time/rate"
)

type Client struct {
	tc        *torrent.Client
	cfg       *config.TorrentConfig
	dataDir   string
	stateMu   sync.RWMutex
	stateData map[string]persistedTorrent
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

	if IsUTPDisabled(cfg) {
		tcConfig.DisableUTP = true
		log.Println("uTP disabled by configuration/environment")
	}

	tc, err := torrent.NewClient(tcConfig)
	if err != nil {
		return nil, err
	}

	client := &Client{
		tc:        tc,
		cfg:       cfg,
		dataDir:   dataDir,
		stateData: make(map[string]persistedTorrent),
	}

	client.loadState()

	go client.startGC()
	go client.startPostProcessor()
	go client.startWatcher()

	return client, nil
}

// IsUTPDisabled checks whether uTP is disabled via configuration or environment variable.
func IsUTPDisabled(cfg *config.TorrentConfig) bool {
	if cfg != nil && cfg.DisableUTP {
		return true
	}
	return os.Getenv("GORRENT_NO_UTP") != ""
}

// AddMagnet adds a torrent via magnet link and starts downloading.
func (c *Client) AddMagnet(magnet string, category string) (*torrent.Torrent, error) {
	return c.AddMagnetWithSeedTime(magnet, category, "")
}

// AddMagnetWithSeedTime adds a torrent via magnet link with an optional custom seed duration.
func (c *Client) AddMagnetWithSeedTime(magnet string, category string, seedTime string) (*torrent.Torrent, error) {
	seedTime = strings.TrimSpace(seedTime)
	if seedTime != "" {
		if _, _, err := ParseSeedDuration(seedTime); err != nil {
			return nil, fmt.Errorf("invalid seed_time: %w", err)
		}
	}

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

	c.stateMu.Lock()
	existing := c.stateData[t.InfoHash().HexString()]
	c.stateData[t.InfoHash().HexString()] = persistedTorrent{
		InfoHash:    t.InfoHash().HexString(),
		Magnet:      magnet,
		Category:    category,
		SeedTime:    seedTime,
		CompletedAt: existing.CompletedAt,
	}
	c.saveState()
	c.stateMu.Unlock()

	return t, nil
}

// SetSeedTime updates the seed limit for an active or persisted torrent.
func (c *Client) SetSeedTime(hash, seedTime string) error {
	seedTime = strings.TrimSpace(seedTime)
	if seedTime != "" {
		if _, _, err := ParseSeedDuration(seedTime); err != nil {
			return fmt.Errorf("invalid seed_time: %w", err)
		}
	}

	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	item, exists := c.stateData[hash]
	if !exists {
		found := false
		for _, t := range c.tc.Torrents() {
			if t.InfoHash().HexString() == hash {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("torrent not found: %s", hash)
		}
		item = persistedTorrent{
			InfoHash: hash,
		}
	}

	item.SeedTime = seedTime
	c.stateData[hash] = item
	c.saveState()
	return nil
}

// StopTorrent drops an active torrent from the client.
func (c *Client) StopTorrent(hash string) error {
	for _, t := range c.tc.Torrents() {
		if t.InfoHash().HexString() == hash {
			t.Drop()

			c.stateMu.Lock()
			delete(c.stateData, hash)
			c.saveState()
			c.stateMu.Unlock()

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
	now := time.Now()

	c.stateMu.RLock()
	stateCopy := make(map[string]persistedTorrent, len(c.stateData))
	for k, v := range c.stateData {
		stateCopy[k] = v
	}
	c.stateMu.RUnlock()

	for _, t := range c.tc.Torrents() {
		info := t.Info()
		name := "Downloading metadata..."
		if info != nil {
			name = info.Name
		}

		stats := t.Stats()
		hash := t.InfoHash().HexString()
		st := map[string]interface{}{
			"hash":       hash,
			"name":       name,
			"downloaded": stats.BytesReadData.Int64(),
			"length":     t.Length(),
			"peers":      stats.ActivePeers,
		}

		if pt, ok := stateCopy[hash]; ok {
			if pt.SeedTime != "" {
				st["seed_time"] = pt.SeedTime
				dur, isInf, err := ParseSeedDuration(pt.SeedTime)
				if err == nil {
					if isInf {
						st["seed_remaining_sec"] = int64(-1)
					} else if pt.CompletedAt > 0 {
						completedTime := time.Unix(pt.CompletedAt, 0)
						deadline := completedTime.Add(dur)
						rem := int64(deadline.Sub(now).Seconds())
						if rem < 0 {
							rem = 0
						}
						st["seed_remaining_sec"] = rem
						st["seed_until"] = deadline.Unix()
					}
				}
			}
			if pt.CompletedAt > 0 {
				st["completed_at"] = pt.CompletedAt
			}
			if pt.Category != "" {
				st["category"] = pt.Category
			}
		}

		statuses = append(statuses, st)
	}
	return statuses
}

// Close shuts down the torrent client.
func (c *Client) Close() {
	c.tc.Close()
	log.Println("Torrent client closed")
}

// ShouldDropTorrent evaluates whether a completed torrent should be cleaned up by GC.
func ShouldDropTorrent(length int64, bytesCompleted int64, bytesWritten int64, pt PersistedTorrent, cfg *config.TorrentConfig, completedTime time.Time, now time.Time) (bool, string) {
	if length == 0 || bytesCompleted < length {
		return false, ""
	}

	// 1. If explicit seed_time is configured for this torrent
	if pt.SeedTime != "" {
		dur, isInf, err := ParseSeedDuration(pt.SeedTime)
		if err == nil {
			if isInf {
				// Infinite seeding: exempt from time-based cleanup
				return false, ""
			}
			if !completedTime.IsZero() && now.Sub(completedTime) >= dur {
				return true, fmt.Sprintf("per-torrent seed_time %s reached", pt.SeedTime)
			}
			// Active per-torrent seed limit not yet reached; do not drop
			return false, ""
		}
	}

	// 2. Global AutoCleanup rules (only when per-torrent override is not set)
	if cfg.AutoCleanup {
		if cfg.SeedRatio > 0 {
			ratio := float64(bytesWritten) / float64(length)
			if ratio >= cfg.SeedRatio {
				return true, fmt.Sprintf("ratio %.2f reached (target: %.2f)", ratio, cfg.SeedRatio)
			}
		}

		if cfg.MaxSeedDays > 0 && !completedTime.IsZero() {
			daysOld := now.Sub(completedTime).Hours() / 24.0
			if daysOld >= float64(cfg.MaxSeedDays) {
				return true, fmt.Sprintf("seeded for %.1f days (max: %d)", daysOld, cfg.MaxSeedDays)
			}
		}
	}

	return false, ""
}

// startGC periodically checks torrents for cleanup conditions.
func (c *Client) startGC() {
	log.Printf("Garbage Collector enabled (Ratio: %.2f, Max Days: %d, AutoCleanup: %v)", c.cfg.SeedRatio, c.cfg.MaxSeedDays, c.cfg.AutoCleanup)
	for {
		time.Sleep(10 * time.Minute)
		now := time.Now()

		c.stateMu.RLock()
		stateCopy := make(map[string]persistedTorrent, len(c.stateData))
		for k, v := range c.stateData {
			stateCopy[k] = v
		}
		c.stateMu.RUnlock()

		for _, t := range c.tc.Torrents() {
			if t.Info() == nil {
				continue // Skip if metadata is not yet downloaded
			}
			length := t.Length()
			if length == 0 || t.BytesCompleted() < length {
				logger.Debugf("GC: Skipping %s (still downloading or empty)", t.InfoHash().HexString())
				continue // Still downloading or empty
			}

			pt := stateCopy[t.InfoHash().HexString()]
			var completedTime time.Time
			if pt.CompletedAt > 0 {
				completedTime = time.Unix(pt.CompletedAt, 0)
			} else {
				path := filepath.Join(c.cfg.DownloadDir, t.Name())
				if stat, err := os.Stat(path); err == nil {
					completedTime = stat.ModTime()
				}
			}

			stats := t.Stats()
			drop, reason := ShouldDropTorrent(length, t.BytesCompleted(), stats.BytesWrittenData.Int64(), pt, c.cfg, completedTime, now)

			if drop {
				log.Printf("GC: Dropping %s (%s)", t.Name(), reason)
				t.Drop()

				c.stateMu.Lock()
				delete(c.stateData, t.InfoHash().HexString())
				c.saveState()
				c.stateMu.Unlock()

				if c.cfg.DeleteFilesOnStop {
					filePath := filepath.Join(c.cfg.DownloadDir, t.Name())
					// Security: ensure the resolved path is actually under DownloadDir
					// to prevent a malicious torrent name from deleting files elsewhere.
					downloadDir := filepath.Clean(c.cfg.DownloadDir)
					if !strings.HasPrefix(filepath.Clean(filePath)+string(filepath.Separator), downloadDir+string(filepath.Separator)) {
						log.Printf("GC: BLOCKED deletion of suspicious path for %s", t.Name())
					} else if err := os.RemoveAll(filePath); err != nil {
						log.Printf("GC: failed to delete files for %s: %v", t.Name(), err)
					} else {
						log.Printf("GC: deleted files for %s", t.Name())
					}
				}
			}
		}
	}
}

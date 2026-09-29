package torrent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type persistentState struct {
	Torrents map[string]PersistedTorrent `json:"torrents"`
}

// PersistedTorrent records metadata and limits for an active or saved torrent.
type PersistedTorrent struct {
	InfoHash    string `json:"info_hash"`
	Magnet      string `json:"magnet"`
	Category    string `json:"category,omitempty"`
	SeedTime    string `json:"seed_time,omitempty"`    // e.g. "2h", "30d", "0" (0 = infinite)
	CompletedAt int64  `json:"completed_at,omitempty"` // Unix epoch seconds when download completed
}

type persistedTorrent = PersistedTorrent

// ParseSeedDuration parses seed time strings like "30d", "2h", "45m", or "0".
// If s == "0", isInfinite is true indicating the torrent should seed indefinitely.
func ParseSeedDuration(s string) (time.Duration, bool, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false, nil
	}
	if s == "0" {
		return 0, true, nil
	}

	// Handle days suffix "d"
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(numStr)
		if err != nil || days < 0 {
			return 0, false, fmt.Errorf("invalid day duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, false, nil
	}

	// Handle bare number of seconds (e.g. "3600")
	if sec, err := strconv.Atoi(s); err == nil {
		if sec < 0 {
			return 0, false, fmt.Errorf("duration cannot be negative: %s", s)
		}
		return time.Duration(sec) * time.Second, false, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false, fmt.Errorf("invalid duration format: %s", s)
	}
	if d < 0 {
		return 0, false, fmt.Errorf("duration cannot be negative: %s", s)
	}
	return d, false, nil
}

func (c *Client) stateFilePath() string {
	return filepath.Join(c.cfg.DownloadDir, "state.json")
}

func (c *Client) loadState() {
	path := c.stateFilePath()
	b, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("State: error reading state.json: %v", err)
		}
		return
	}

	var s persistentState
	if err := json.Unmarshal(b, &s); err != nil {
		log.Printf("State: error parsing state.json: %v", err)
		return
	}

	c.stateMu.Lock()
	c.stateData = s.Torrents
	if c.stateData == nil {
		c.stateData = make(map[string]persistedTorrent)
	}
	c.stateMu.Unlock()

	loadedCount := 0
	for _, t := range c.stateData {
		loadedCount++
		// Re-add torrent in the background to not block boot
		go func(pt persistedTorrent) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("State: panic recovering torrent %s: %v", pt.InfoHash, r)
				}
			}()
			log.Printf("State: recovering torrent %s...", pt.InfoHash)
			_, err := c.AddMagnetWithSeedTime(pt.Magnet, pt.Category, pt.SeedTime)
			if err != nil {
				log.Printf("State: failed to recover %s: %v", pt.InfoHash, err)
			}
		}(t)
	}

	if loadedCount > 0 {
		log.Printf("State: self-healing boot initiated for %d torrents", loadedCount)
	}
}

func (c *Client) saveState() {
	path := c.stateFilePath()

	c.stateMu.RLock()
	s := persistentState{
		Torrents: c.stateData,
	}
	b, err := json.MarshalIndent(s, "", "  ")
	c.stateMu.RUnlock()

	if err != nil {
		log.Printf("State: failed to serialize: %v", err)
		return
	}

	if err := os.WriteFile(path, b, 0644); err != nil {
		log.Printf("State: failed to write state.json: %v", err)
	}
}

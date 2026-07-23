package torrent

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type persistentState struct {
	Torrents map[string]persistedTorrent `json:"torrents"`
}

type persistedTorrent struct {
	InfoHash string `json:"info_hash"`
	Magnet   string `json:"magnet"`
	Category string `json:"category"`
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
			_, err := c.AddMagnet(pt.Magnet, pt.Category)
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

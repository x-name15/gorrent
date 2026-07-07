package torrent

import (
	"fmt"
	"log"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/x-name15/gorrent/pkg/config"
)

type Client struct {
	tc  *torrent.Client
	cfg *config.TorrentConfig
}

// NewClient initializes the anacrolix/torrent client.
func NewClient(cfg *config.TorrentConfig) (*Client, error) {
	tcConfig := torrent.NewDefaultClientConfig()
	tcConfig.DataDir = cfg.DownloadDir
	// Disable seed by default for now to save bandwidth, unless requested
	tcConfig.NoUpload = false

	tc, err := torrent.NewClient(tcConfig)
	if err != nil {
		return nil, err
	}

	return &Client{
		tc:  tc,
		cfg: cfg,
	}, nil
}

// AddMagnet adds a torrent via magnet link and starts downloading.
func (c *Client) AddMagnet(magnet string) (*torrent.Torrent, error) {
	t, err := c.tc.AddMagnet(magnet)
	if err != nil {
		return nil, err
	}

	// Wait for info to download with a 30-second timeout
	select {
	case <-t.GotInfo():
		// Info downloaded, we can proceed
	case <-time.After(30 * time.Second):
		t.Drop() // Clean up resources
		return nil, fmt.Errorf("timeout waiting for torrent metadata (0 seeders or dead torrent)")
	}

	// Download all files
	t.DownloadAll()

	return t, nil
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
			"downloaded": stats.BytesReadData,
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

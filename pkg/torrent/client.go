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

package torrent

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/x-name15/gorrent/pkg/logger"
)

const (
	watchPollInterval = 5 * time.Second
	// Maximum bytes read from a watch file — any valid magnet URI fits in 2 KB.
	watchMaxReadBytes = 2 * 1024
)

// startWatcher polls WatchDir every 5 seconds for .magnet or .txt files containing
// a magnet URI or bare infohash. Handled files are moved to WatchDir/handled/
// before AddMagnet is called, so a crash between the two can't cause a double-download
// (anacrolix/torrent deduplicates by hash anyway).
func (c *Client) startWatcher() {
	if c.cfg.WatchDir == "" {
		return
	}

	watchDir := filepath.Clean(c.cfg.WatchDir)
	handledDir := filepath.Join(watchDir, "handled")
	os.MkdirAll(handledDir, 0755)
	log.Printf("Watch Folder active: %s", watchDir)

	for {
		logger.Debugf("Watcher: Polling %s for new magnets...", watchDir)
		entries, err := os.ReadDir(watchDir)
		if err != nil {
			log.Printf("Watcher: error reading dir %s: %v", watchDir, err)
			time.Sleep(watchPollInterval)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".magnet" && ext != ".txt" {
				continue
			}

			srcPath := filepath.Join(watchDir, name)

			// Security: ensure the file is actually inside watchDir
			// (guards against symlink attacks pointing outside the directory)
			if !strings.HasPrefix(filepath.Clean(srcPath)+string(filepath.Separator), watchDir+string(filepath.Separator)) {
				log.Printf("Watcher: BLOCKED suspicious path %s", srcPath)
				continue
			}

			dstPath := filepath.Join(handledDir, name)

			// Open and read at most watchMaxReadBytes — protects against huge files in RAM
			f, err := os.Open(srcPath)
			if err != nil {
				log.Printf("Watcher: error opening %s: %v", name, err)
				continue
			}
			b, err := io.ReadAll(io.LimitReader(f, watchMaxReadBytes))
			f.Close()
			if err != nil {
				log.Printf("Watcher: error reading %s: %v", name, err)
				continue
			}

			magnet := strings.TrimSpace(string(b))
			if magnet == "" {
				continue
			}

			// Move to handled/ BEFORE AddMagnet so a crash can't cause a double-download.
			// anacrolix/torrent deduplicates by hash, so adding twice is harmless anyway.
			if err := os.Rename(srcPath, dstPath); err != nil {
				log.Printf("Watcher: could not archive %s: %v", name, err)
				continue
			}

			log.Printf("Watcher: picked up %s", name)
			if _, err = c.AddMagnet(magnet, ""); err != nil {
				log.Printf("Watcher: failed to add %s: %v", name, err)
			}
		}

		time.Sleep(watchPollInterval)
	}
}

package torrent

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
)

// startPostProcessor handles running scripts, creating hardlinks, and generating playlists when torrents finish.
func (c *Client) startPostProcessor() {
	playlistEnabled := !c.cfg.NoPlaylist && os.Getenv("GORRENT_NO_PLAYLIST") == ""
	log.Printf("Post-Processor active (Playlists: %v, Hardlinks: %v, PostScript: %v)", playlistEnabled, c.cfg.HardlinkDir != "", c.cfg.PostScript != "")

	os.MkdirAll(c.dataDir, 0755)
	stateFile := filepath.Join(c.dataDir, "gorrent_processed.json")
	processed := make(map[string]bool)

	// Load existing state
	if b, err := os.ReadFile(stateFile); err == nil {
		json.Unmarshal(b, &processed)
	}

	for {
		time.Sleep(1 * time.Minute)
		for _, t := range c.tc.Torrents() {
			if t.Info() == nil {
				continue
			}

			hash := t.InfoHash().HexString()
			if processed[hash] {
				continue
			}

			length := t.Length()
			if length > 0 && t.BytesCompleted() == length {
				log.Printf("Post-Processing %s", t.Name())

				category, targetPath := c.getTorrentCategoryAndPath(t)

				// 1. Record completion timestamp in stateData if not already set
				c.stateMu.Lock()
				if item, exists := c.stateData[hash]; exists && item.CompletedAt == 0 {
					item.CompletedAt = time.Now().Unix()
					c.stateData[hash] = item
					c.saveState()
				}
				c.stateMu.Unlock()

				// 2. Playlist generation for multi-file audio/video torrents
				if playlistEnabled {
					var filePaths []string
					for _, f := range t.Files() {
						filePaths = append(filePaths, f.Path())
					}
					baseDir := filepath.Dir(targetPath)
					if !strings.HasSuffix(filepath.Clean(targetPath), filepath.Clean(t.Name())) {
						baseDir = targetPath
					}
					_ = WritePlaylists(baseDir, filePaths)
				}

				// 3. Hardlinks
				if c.cfg.HardlinkDir != "" {
					c.createHardlinks(t, targetPath, category)
				}

				// 4. Post-script
				if c.cfg.PostScript != "" {
					c.runPostScript(t, targetPath, category)
				}

				processed[hash] = true
				b, _ := json.MarshalIndent(processed, "", "  ")
				os.WriteFile(stateFile, b, 0644)
			}
		}
	}
}

// getTorrentCategoryAndPath resolves the absolute path and category where the torrent was saved.
func (c *Client) getTorrentCategoryAndPath(t *torrent.Torrent) (string, string) {
	name := t.Name()

	for catName, catDir := range c.cfg.CategoryDirs {
		p := filepath.Join(catDir, name)
		if _, err := os.Stat(p); err == nil {
			return catName, p
		}
	}

	return "", filepath.Join(c.cfg.DownloadDir, name)
}

func (c *Client) createHardlinks(t *torrent.Torrent, srcRoot string, category string) {
	destRoot := filepath.Join(c.cfg.HardlinkDir, category, t.Name())
	// Security: ensure destRoot is actually under HardlinkDir (prevent path traversal from malicious torrent names)
	if !strings.HasPrefix(filepath.Clean(destRoot)+string(filepath.Separator), filepath.Clean(c.cfg.HardlinkDir)+string(filepath.Separator)) {
		log.Printf("Hardlink BLOCKED: suspicious torrent name '%s' would escape HardlinkDir", t.Name())
		return
	}

	err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return nil
		}

		destPath := filepath.Join(destRoot, relPath)
		os.MkdirAll(filepath.Dir(destPath), 0755)

		if err := os.Link(path, destPath); err != nil {
			if !os.IsExist(err) {
				log.Printf("Failed to hardlink %s -> %s: %v", path, destPath, err)
			}
		} else {
			log.Printf("Hardlinked: %s", destPath)
		}
		return nil
	})

	if err != nil {
		log.Printf("Hardlink walk error for %s: %v", t.Name(), err)
	}
}

func (c *Client) runPostScript(t *torrent.Torrent, srcRoot string, category string) {
	cmd := exec.Command(c.cfg.PostScript)
	cmd.Env = append(os.Environ(),
		"GORRENT_HASH="+t.InfoHash().HexString(),
		"GORRENT_NAME="+t.Name(),
		"GORRENT_PATH="+srcRoot,
		"GORRENT_CATEGORY="+category,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("PostScript failed for %s: %v\nOutput: %s", t.Name(), err, string(out))
	} else {
		log.Printf("PostScript success for %s", t.Name())
	}
}

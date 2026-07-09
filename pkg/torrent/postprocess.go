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

// startPostProcessor handles running scripts and creating hardlinks when torrents finish.
func (c *Client) startPostProcessor() {
	if c.cfg.HardlinkDir == "" && c.cfg.PostScript == "" {
		return
	}

	log.Printf("Post-Processor enabled. Hardlinks: %s | PostScript: %s", c.cfg.HardlinkDir, c.cfg.PostScript)
	if c.cfg.PostScript != "" {
		log.Printf("[NOTE] post_script runs the script as a native process. If running inside Docker (scratch image), use the 'callback' webhook instead.")
	}

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

				if c.cfg.HardlinkDir != "" {
					c.createHardlinks(t, targetPath, category)
				}

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

package torrent

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

var defaultAnnounceTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://exodus.desync.com:6969/announce",
	"udp://open.stealth.si:80/announce",
	"udp://tracker.dler.org:6969/announce",
}

// SeedResult describes a newly created and seeded local torrent.
type SeedResult struct {
	InfoHash    string `json:"info_hash"`
	Name        string `json:"name"`
	TotalBytes  int64  `json:"total_bytes"`
	Magnet      string `json:"magnet"`
	TorrentFile string `json:"torrent_file,omitempty"`
}

// BuildSeedMetaInfo generates the .torrent metainfo, magnet URI, and writes the .torrent file
// to downloadDir (if non-empty) for an existing local file or directory.
func BuildSeedMetaInfo(localPath string, extraTrackers []string, downloadDir string) (*metainfo.MetaInfo, *metainfo.Info, string, string, error) {
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	fi, err := os.Stat(absPath)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("path does not exist or cannot be accessed: %w", err)
	}

	var totalSize int64
	if fi.IsDir() {
		err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				info, err := d.Info()
				if err == nil {
					totalSize += info.Size()
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to walk directory: %w", err)
		}
	} else {
		totalSize = fi.Size()
	}

	pieceLen := metainfo.ChoosePieceLength(totalSize)
	if pieceLen <= 0 {
		pieceLen = 256 * 1024
	}

	info := metainfo.Info{
		PieceLength: pieceLen,
	}

	if err := info.BuildFromFilePath(absPath); err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to build torrent metainfo: %w", err)
	}

	var announceList metainfo.AnnounceList
	seen := make(map[string]bool)

	addTracker := func(tr string) {
		tr = strings.TrimSpace(tr)
		if tr != "" && !seen[tr] {
			seen[tr] = true
			announceList = append(announceList, []string{tr})
		}
	}

	for _, tr := range extraTrackers {
		addTracker(tr)
	}
	for _, tr := range defaultAnnounceTrackers {
		addTracker(tr)
	}

	mi := metainfo.MetaInfo{
		AnnounceList: announceList,
	}
	mi.SetDefaults()

	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to bencode info dictionary: %w", err)
	}
	mi.InfoBytes = infoBytes

	infoHash := mi.HashInfoBytes()
	mag := mi.Magnet(&infoHash, &info)
	magnetURI := mag.String()

	torrentFilePath := ""
	if downloadDir != "" {
		torrentFilePath = filepath.Join(downloadDir, info.Name+".torrent")
		if f, err := os.Create(torrentFilePath); err == nil {
			_ = mi.Write(f)
			_ = f.Close()
		}
	}

	return &mi, &info, magnetURI, torrentFilePath, nil
}

// SeedPath turns an existing local file or directory into a torrent,
// saves the .torrent file to download_dir, begins seeding it to peers,
// and returns the seed result including magnet URI.
func (c *Client) SeedPath(localPath string, category string) (*SeedResult, error) {
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	mi, info, magnetURI, torrentFilePath, err := BuildSeedMetaInfo(absPath, c.cfg.Trackers, c.cfg.DownloadDir)
	if err != nil {
		return nil, err
	}

	spec := torrent.TorrentSpecFromMetaInfo(mi)
	spec.Storage = storage.NewFile(filepath.Dir(absPath))

	t, _, err := c.tc.AddTorrentSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent to client: %w", err)
	}

	t.DownloadAll()

	infoHashHex := mi.HashInfoBytes().HexString()
	c.stateMu.Lock()
	c.stateData[infoHashHex] = persistedTorrent{
		InfoHash: infoHashHex,
		Magnet:   magnetURI,
		Category: category,
	}
	c.saveState()
	c.stateMu.Unlock()

	return &SeedResult{
		InfoHash:    infoHashHex,
		Name:        info.Name,
		TotalBytes:  info.TotalLength(),
		Magnet:      magnetURI,
		TorrentFile: torrentFilePath,
	}, nil
}

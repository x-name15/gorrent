package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/torrent"
)

func TestBuildSeedMetaInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorrent-seed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	downloadDir := filepath.Join(tempDir, "downloads")
	seedTargetDir := filepath.Join(tempDir, "my-music-album")

	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		t.Fatalf("failed to create download dir: %v", err)
	}
	if err := os.MkdirAll(seedTargetDir, 0755); err != nil {
		t.Fatalf("failed to create seed dir: %v", err)
	}

	content1 := []byte("Hello BitTorrent Seeder Track 1!")
	content2 := []byte("Hello BitTorrent Seeder Track 2! Extra bytes for testing.")

	sampleFile1 := filepath.Join(seedTargetDir, "track1.mp3")
	sampleFile2 := filepath.Join(seedTargetDir, "track2.mp3")
	if err := os.WriteFile(sampleFile1, content1, 0644); err != nil {
		t.Fatalf("failed to write sample file 1: %v", err)
	}
	if err := os.WriteFile(sampleFile2, content2, 0644); err != nil {
		t.Fatalf("failed to write sample file 2: %v", err)
	}

	expectedTotalSize := int64(len(content1) + len(content2))
	extraTrackers := []string{"udp://custom.tracker.org:1337/announce"}

	mi, info, magnetURI, torrentFilePath, err := torrent.BuildSeedMetaInfo(seedTargetDir, extraTrackers, downloadDir)
	if err != nil {
		t.Fatalf("BuildSeedMetaInfo failed: %v", err)
	}

	infoHashHex := mi.HashInfoBytes().HexString()
	if len(infoHashHex) != 40 {
		t.Errorf("expected 40-char infohash, got %d (%s)", len(infoHashHex), infoHashHex)
	}
	if !strings.HasPrefix(magnetURI, "magnet:?xt=urn:btih:") {
		t.Errorf("expected valid magnet link, got: %s", magnetURI)
	}
	if !strings.Contains(magnetURI, "custom.tracker.org") {
		t.Errorf("expected custom tracker in magnet URI: %s", magnetURI)
	}
	if info.Name != "my-music-album" {
		t.Errorf("expected name 'my-music-album', got %s", info.Name)
	}
	if info.TotalLength() != expectedTotalSize {
		t.Errorf("expected exact total length %d, got %d", expectedTotalSize, info.TotalLength())
	}

	// Verify .torrent file was created on disk and has non-zero size
	fi, err := os.Stat(torrentFilePath)
	if err != nil {
		t.Errorf("expected torrent file at %s, got error: %v", torrentFilePath, err)
	} else if fi.Size() == 0 {
		t.Errorf("expected non-empty .torrent file, got size 0")
	}
}

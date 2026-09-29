package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/torrent"
)

func TestPlaylist_NaturalSorting(t *testing.T) {
	files := []string{
		"track10.mp3",
		"track1.mp3",
		"track20.mp3",
		"track2.mp3",
		"track03.mp3",
	}

	// Test NaturalLess
	expectedOrder := []string{
		"track1.mp3",
		"track2.mp3",
		"track03.mp3",
		"track10.mp3",
		"track20.mp3",
	}

	for i := 0; i < len(expectedOrder)-1; i++ {
		for j := i + 1; j < len(expectedOrder); j++ {
			a := expectedOrder[i]
			b := expectedOrder[j]
			if !torrent.NaturalLess(a, b) {
				t.Fatalf("expected NaturalLess(%q, %q) to be true", a, b)
			}
			if torrent.NaturalLess(b, a) {
				t.Fatalf("expected NaturalLess(%q, %q) to be false", b, a)
			}
		}
	}

	_ = files
}

func TestPlaylist_BuildPlaylists(t *testing.T) {
	inputFiles := []string{
		// Folder "Album1" has 3 media files + 1 non-media file
		"Album1/track02.flac",
		"Album1/track01.flac",
		"Album1/track10.flac",
		"Album1/cover.jpg",
		"Album1/info.nfo",

		// Folder "Album2" has only 1 media file -> should NOT produce a playlist
		"Album2/solo.mp3",
		"Album2/notes.txt",

		// Folder "Movies/Season1" has 2 video files
		"Movies/Season1/episode2.mkv",
		"Movies/Season1/episode1.mkv",

		// Root media file (only 1)
		"trailer.mp4",
	}

	playlists := torrent.BuildPlaylists(inputFiles)

	// Check Album1 playlist
	album1M3U, ok := playlists["Album1/playlist.m3u"]
	if !ok {
		t.Fatalf("expected playlist for Album1/playlist.m3u, got none")
	}
	lines := strings.Split(strings.TrimSpace(album1M3U), "\n")
	if len(lines) != 4 { // #EXTM3U + 3 tracks
		t.Fatalf("expected 4 lines in Album1 playlist, got %d:\n%s", len(lines), album1M3U)
	}
	if lines[0] != "#EXTM3U" {
		t.Errorf("expected #EXTM3U header, got %s", lines[0])
	}
	if lines[1] != "./track01.flac" || lines[2] != "./track02.flac" || lines[3] != "./track10.flac" {
		t.Errorf("unexpected track order in Album1:\n%s", album1M3U)
	}

	// Check Album2 was skipped (only 1 track)
	if _, ok := playlists["Album2/playlist.m3u"]; ok {
		t.Errorf("Album2 has only 1 track and should not have a playlist generated")
	}

	// Check Season1 playlist
	season1M3U, ok := playlists["Movies/Season1/playlist.m3u"]
	if !ok {
		t.Fatalf("expected playlist for Movies/Season1/playlist.m3u, got none")
	}
	sLines := strings.Split(strings.TrimSpace(season1M3U), "\n")
	if len(sLines) != 3 { // #EXTM3U + 2 tracks
		t.Fatalf("expected 3 lines in Movies/Season1 playlist, got %d", len(sLines))
	}
	if sLines[1] != "./episode1.mkv" || sLines[2] != "./episode2.mkv" {
		t.Errorf("unexpected video order in Movies/Season1:\n%s", season1M3U)
	}

	// Root single file should not generate playlist
	if _, ok := playlists["playlist.m3u"]; ok {
		t.Errorf("root with 1 file should not have generated a playlist")
	}
}

func TestPlaylist_PathTraversalGuard(t *testing.T) {
	// Attempted traversal filenames
	maliciousFiles := []string{
		"../../etc/passwd.mp3",
		"../outside.mp3",
		"/absolute/path/file.mp3",
	}

	playlists := torrent.BuildPlaylists(maliciousFiles)
	if len(playlists) > 0 {
		t.Errorf("expected traversal paths to be rejected, got %v", playlists)
	}
}

func TestPlaylist_WritePlaylists_ZeroOverwrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorrent-playlist-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	albumDir := filepath.Join(tempDir, "Album")
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		t.Fatalf("failed to create album dir: %v", err)
	}

	// Pre-create an existing custom playlist.m3u
	existingContent := "#EXTM3U\n# Custom curated playlist\n./special.mp3\n"
	existingPath := filepath.Join(albumDir, "playlist.m3u")
	if err := os.WriteFile(existingPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing playlist: %v", err)
	}

	files := []string{
		"Album/song1.mp3",
		"Album/song2.mp3",
		"Album/song3.mp3",
	}

	// Call WritePlaylists
	if err := torrent.WritePlaylists(tempDir, files); err != nil {
		t.Fatalf("WritePlaylists failed: %v", err)
	}

	// Verify that the existing playlist.m3u was NOT overwritten
	actualContent, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("failed to read playlist file: %v", err)
	}
	if string(actualContent) != existingContent {
		t.Fatalf("existing playlist was overwritten! Expected:\n%s\nGot:\n%s", existingContent, string(actualContent))
	}
}

func TestPlaylist_WritePlaylists_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorrent-playlist-create-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	albumDir := filepath.Join(tempDir, "Music")
	if err := os.MkdirAll(albumDir, 0755); err != nil {
		t.Fatalf("failed to create music dir: %v", err)
	}

	files := []string{
		"Music/track2.ogg",
		"Music/track1.ogg",
	}

	if err := torrent.WritePlaylists(tempDir, files); err != nil {
		t.Fatalf("WritePlaylists failed: %v", err)
	}

	playlistPath := filepath.Join(albumDir, "playlist.m3u")
	data, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("playlist.m3u was not created: %v", err)
	}

	expected := "#EXTM3U\n./track1.ogg\n./track2.ogg\n"
	// Normalize line endings for Windows/Linux
	actual := strings.ReplaceAll(string(data), "\r\n", "\n")
	if actual != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, actual)
	}
}

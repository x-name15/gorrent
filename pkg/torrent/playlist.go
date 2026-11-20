package torrent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var mediaExtensions = map[string]bool{
	".aac":  true,
	".aiff": true,
	".avi":  true,
	".flac": true,
	".flv":  true,
	".m2ts": true,
	".m4a":  true,
	".m4v":  true,
	".mka":  true,
	".mkv":  true,
	".mov":  true,
	".mp3":  true,
	".mp4":  true,
	".mpeg": true,
	".mpg":  true,
	".oga":  true,
	".ogg":  true,
	".ogv":  true,
	".opus": true,
	".wav":  true,
	".webm": true,
	".wma":  true,
	".wmv":  true,
}

var winDriveRegex = regexp.MustCompile(`^[a-zA-Z]:`)

// NaturalLess implements alphanumeric natural comparison so "track2" sorts before "track10".
func NaturalLess(a, b string) bool {
	aChunks := splitIntoChunks(a)
	bChunks := splitIntoChunks(b)

	for i := 0; i < len(aChunks) && i < len(bChunks); i++ {
		ca, cb := aChunks[i], bChunks[i]
		isANum := isAllDigits(ca)
		isBNum := isAllDigits(cb)

		if isANum && isBNum {
			na, errA := strconv.ParseUint(ca, 10, 64)
			nb, errB := strconv.ParseUint(cb, 10, 64)
			if errA == nil && errB == nil {
				if na != nb {
					return na < nb
				}
				// Same numeric value; compare string lengths (for leading zeros)
				if len(ca) != len(cb) {
					return len(ca) < len(cb)
				}
			}
		}

		lowerA := strings.ToLower(ca)
		lowerB := strings.ToLower(cb)
		if lowerA != lowerB {
			return lowerA < lowerB
		}
	}
	return len(aChunks) < len(bChunks)
}

func splitIntoChunks(s string) []string {
	var chunks []string
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return chunks
	}

	start := 0
	isDigit := unicode.IsDigit(runes[0])

	for i := 1; i < n; i++ {
		curDigit := unicode.IsDigit(runes[i])
		if curDigit != isDigit {
			chunks = append(chunks, string(runes[start:i]))
			start = i
			isDigit = curDigit
		}
	}
	chunks = append(chunks, string(runes[start:n]))
	return chunks
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// BuildPlaylists analyzes relative file paths and returns a map of relative playlist path -> M3U content.
// Only directories containing 2 or more audio/video media files will receive a playlist.
func BuildPlaylists(filePaths []string) map[string]string {
	groups := make(map[string][]string)
	occupied := make(map[string]bool)

	for _, raw := range filePaths {
		file := filepath.ToSlash(strings.TrimSpace(raw))
		if file == "" {
			continue
		}

		// Security checks: reject Windows drive letters, path traversal, control chars
		if winDriveRegex.MatchString(file) || strings.HasPrefix(file, "/") || strings.Contains(file, "\\") {
			continue
		}
		parts := strings.Split(file, "/")
		hasInvalidPart := false
		for _, p := range parts {
			if p == "" || p == "." || p == ".." || strings.ContainsAny(p, "\x00\r\n\t") {
				hasInvalidPart = true
				break
			}
		}
		if hasInvalidPart {
			continue
		}

		occupied[strings.ToLower(file)] = true

		ext := strings.ToLower(filepath.Ext(file))
		if !mediaExtensions[ext] {
			continue
		}

		// Group under each ancestor directory
		for i := 1; i < len(parts); i++ {
			dir := strings.Join(parts[:i], "/")
			rel := strings.Join(parts[i:], "/")
			groups[dir] = append(groups[dir], rel)
		}
	}

	playlists := make(map[string]string)
	for dir, entries := range groups {
		target := dir + "/playlist.m3u"
		// Never overwrite an existing playlist supplied by the torrent or folder with < 2 items
		if len(entries) < 2 || occupied[strings.ToLower(target)] {
			continue
		}

		// Sort naturally
		sort.Slice(entries, func(i, j int) bool {
			return NaturalLess(entries[i], entries[j])
		})

		var sb strings.Builder
		sb.WriteString("#EXTM3U\n")
		for _, entry := range entries {
			sb.WriteString("./")
			sb.WriteString(entry)
			sb.WriteString("\n")
		}
		playlists[target] = sb.String()
	}

	return playlists
}

// WritePlaylists writes generated playlists to disk inside targetDir.
// It never overwrites existing playlists (uses os.O_CREATE|os.O_EXCL).
func WritePlaylists(targetDir string, filePaths []string) error {
	playlists := BuildPlaylists(filePaths)
	if len(playlists) == 0 {
		return nil
	}

	absRoot, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for targetDir: %w", err)
	}

	cleanRoot := filepath.Clean(absRoot)

	for relTarget, content := range playlists {
		fullTarget := filepath.Join(cleanRoot, filepath.FromSlash(relTarget))
		parentDir := filepath.Dir(fullTarget)

		// Security: ensure parent stays under cleanRoot
		cleanParent := filepath.Clean(parentDir)
		if !strings.HasPrefix(cleanParent+string(filepath.Separator), cleanRoot+string(filepath.Separator)) && cleanParent != cleanRoot {
			continue
		}

		// Check if parent directory exists; do not create missing torrent directories
		if fi, err := os.Stat(cleanParent); err != nil || !fi.IsDir() {
			continue
		}

		// Exclusive write: never overwrite user or existing playlist
		f, err := os.OpenFile(fullTarget, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			if os.IsExist(err) {
				continue // file already exists, ignore safely
			}
			continue
		}
		_, _ = f.WriteString(content)
		_ = f.Close()
	}

	return nil
}

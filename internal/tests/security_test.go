package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPathTraversalGuard(t *testing.T) {
	// Simulate the guard logic used in watcher and GC
	baseDir := filepath.Clean("/downloads/watch")

	outsideAbs, _ := filepath.Abs("../../../etc/shadow")

	tests := []struct {
		name       string
		fileName   string
		shouldPass bool
	}{
		{"Normal file", "magnet.txt", true},
		{"Subdirectory file", "subdir/magnet.txt", true},
		{"Path traversal back", "../magnet.txt", false},
		{"Path traversal deep", "../../etc/passwd", false},
		{"Deceptive name matching prefix", "../watch2/magnet.txt", false},
		{"Absolute path outside", outsideAbs, false},
		{"Absolute path inside", filepath.Join(baseDir, "magnet.txt"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcPath := filepath.Join(baseDir, tt.fileName)
			if filepath.IsAbs(tt.fileName) {
				srcPath = filepath.Clean(tt.fileName)
			}

			// The exact logic from watcher.go / client.go
			hasPrefix := strings.HasPrefix(filepath.Clean(srcPath)+string(filepath.Separator), baseDir+string(filepath.Separator))

			if hasPrefix != tt.shouldPass {
				t.Errorf("expected pass=%v, got %v for path %s", tt.shouldPass, hasPrefix, tt.fileName)
			}
		})
	}
}

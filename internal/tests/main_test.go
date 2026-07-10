package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/netutil"
)

func TestDisableDirListing(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	
	// Create a file
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Set up the file server with the disableDirListing wrapper
	fs := http.FileServer(netutil.DisableDirListing{FS: http.Dir(tmpDir)})
	
	// Test 1: Accessing the root directory should fail (404)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	fs.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for directory root, got %v", rr.Code)
	}

	// Test 2: Accessing the subdirectory should fail (404)
	req = httptest.NewRequest("GET", "/subdir/", nil)
	rr = httptest.NewRecorder()
	fs.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for subdirectory, got %v", rr.Code)
	}

	// Test 3: Accessing the actual file should succeed (200)
	req = httptest.NewRequest("GET", "/test.txt", nil)
	rr = httptest.NewRecorder()
	fs.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 for file, got %v", rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	if strings.TrimSpace(string(body)) != "hello" {
		t.Errorf("Expected body 'hello', got '%s'", body)
	}
}

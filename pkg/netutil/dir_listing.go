package netutil

import (
	"net/http"
	"os"
)

// DisableDirListing wraps an http.FileSystem to return 404 for directories,
// preventing exposure of the entire directory structure over HTTP endpoints.
type DisableDirListing struct {
	FS http.FileSystem
}

func (d DisableDirListing) Open(path string) (http.File, error) {
	f, err := d.FS.Open(path)
	if err != nil {
		return nil, err
	}
	s, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if s.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

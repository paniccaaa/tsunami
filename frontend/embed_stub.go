//go:build !embed_frontend

/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package frontend

import (
	"io/fs"
	"testing/fstest"
)

// GetFS returns an empty filesystem when frontend is not embedded.
func GetFS() (fs.FS, error) {
	return fstest.MapFS{}, nil
}

// IsEmbedded returns false when frontend is not embedded.
func IsEmbedded() bool {
	return false
}

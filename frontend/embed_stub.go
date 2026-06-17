//go:build !embed_frontend

package frontend

import (
	"io/fs"
	"testing/fstest"
)

func GetFS() (fs.FS, error) {
	return fstest.MapFS{}, nil
}

func IsEmbedded() bool {
	return false
}

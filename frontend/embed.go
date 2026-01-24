//go:build embed_frontend

/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package frontend

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// GetFS returns the embedded frontend filesystem.
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// IsEmbedded returns true when frontend is embedded.
func IsEmbedded() bool {
	return true
}

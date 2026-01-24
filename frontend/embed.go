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
// The dist folder contains the built React application.
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

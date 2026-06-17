//go:build embed_frontend

package frontend

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

func IsEmbedded() bool {
	return true
}

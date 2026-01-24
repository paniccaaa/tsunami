/*
Copyright © 2025 Semen Adamenko <semaadamenko1@gmail.com>
*/
package server

import (
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler serves static files with SPA fallback.
// For any request that doesn't match a static file, it serves index.html
// to support client-side routing.
type SPAHandler struct {
	staticFS   fs.FS
	fileServer http.Handler
}

// NewSPAHandler creates a new SPA handler with the given filesystem.
func NewSPAHandler(staticFS fs.FS) *SPAHandler {
	return &SPAHandler{
		staticFS:   staticFS,
		fileServer: http.FileServer(http.FS(staticFS)),
	}
}

// ServeHTTP implements http.Handler.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Remove leading slash for fs.Open
	fsPath := strings.TrimPrefix(path, "/")
	if fsPath == "" {
		fsPath = "index.html"
	}

	// Try to open the file
	f, err := h.staticFS.Open(fsPath)
	if err != nil {
		// File not found - serve index.html for SPA routing
		r.URL.Path = "/"
		h.fileServer.ServeHTTP(w, r)
		return
	}
	f.Close()

	// File exists - serve it
	h.fileServer.ServeHTTP(w, r)
}

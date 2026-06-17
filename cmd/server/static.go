package server

import (
	"io/fs"
	"net/http"
	"strings"
)

type SPAHandler struct {
	staticFS   fs.FS
	fileServer http.Handler
}

func NewSPAHandler(staticFS fs.FS) *SPAHandler {
	return &SPAHandler{
		staticFS:   staticFS,
		fileServer: http.FileServer(http.FS(staticFS)),
	}
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	fsPath := strings.TrimPrefix(path, "/")
	if fsPath == "" {
		fsPath = "index.html"
	}

	f, err := h.staticFS.Open(fsPath)
	if err != nil {
		r.URL.Path = "/"
		h.fileServer.ServeHTTP(w, r)
		return
	}
	f.Close()

	h.fileServer.ServeHTTP(w, r)
}

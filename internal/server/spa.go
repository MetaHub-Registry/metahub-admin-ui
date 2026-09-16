package server

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serves the embedded SPA: real files when they exist,
// index.html otherwise (client-side routing fallback).
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.opts.Assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mounted via r.Handle("/*") — the full request path is in URL.Path.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if f, err := s.opts.Assets.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		serveIndex(w, s.opts.Assets)
	})
}

// serveIndex writes the SPA entry point (index.html) with a 200 status.
func serveIndex(w http.ResponseWriter, assets fs.FS) {
	f, err := assets.Open("index.html")
	if err != nil {
		// Placeholder/build missing — degrade loudly but never 5xx; this only
		// affects the SPA mount, the /api/v1 surface is unaffected.
		http.Error(w, "SPA assets missing: run `npm run build`", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

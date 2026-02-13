package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed all:static
var staticFS embed.FS

func (s *Server) registerStaticRoutes() {
	// Check if embedded static files exist (ignore .gitkeep placeholder)
	_, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		slog.Info("No embedded static files found, SPA serving disabled")
		s.mux.HandleFunc("GET /{path...}", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{
				"message": "TiUP Visualizer API - Frontend not built yet",
			})
		})
		return
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		slog.Error("Failed to create static sub-filesystem", "error", err)
		return
	}

	fileServer := http.FileServer(http.FS(staticSub))

	// Serve SPA: catch-all for any path that doesn't match API/WS/health
	s.mux.HandleFunc("GET /{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip API, WS, and health routes (already registered)
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws/") || path == "/health" {
			http.NotFound(w, r)
			return
		}

		// Try to serve the exact file first
		if path != "/" {
			cleanPath := strings.TrimPrefix(path, "/")
			if f, err := staticSub.Open(cleanPath); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fallback: serve index.html for SPA routing
		indexData, err := staticFS.ReadFile("static/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexData)
	})
}

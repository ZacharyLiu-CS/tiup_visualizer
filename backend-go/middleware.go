package main

import (
	"net/http"
	"strings"
)

// CORSMiddleware adds CORS headers to responses.
func CORSMiddleware(origins []string, next http.Handler) http.Handler {
	originSet := make(map[string]bool)
	allowAll := false
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowAll || originSet[origin] || len(origins) == 0 {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs incoming requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for static assets
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

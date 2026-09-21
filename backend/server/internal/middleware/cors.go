package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORSOptions holds configuration for the CORS middleware.
type CORSOptions struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int // seconds
}

// DefaultCORSOptions returns sensible defaults for development.
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"chrome-extension://*",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"X-User-ID",
			"X-Request-ID",
		},
		MaxAge: 86400, // 24 hours
	}
}

// CORS returns an HTTP middleware that sets CORS headers based on the
// provided options. Requests with origins not in the allowed list are
// rejected.
func CORS(opts CORSOptions) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool, len(opts.AllowedOrigins))
	for _, o := range opts.AllowedOrigins {
		allowedMap[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if the origin is allowed
			allowed := false
			if allowedMap[origin] || allowedMap["*"] {
				allowed = true
			}
			// Allow chrome-extension://* origins
			if strings.HasPrefix(origin, "chrome-extension://") {
				for _, o := range opts.AllowedOrigins {
					if o == "chrome-extension://*" {
						allowed = true
						break
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(opts.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(opts.AllowedHeaders, ", "))
				if opts.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", opts.MaxAge))
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				if allowed {
					w.WriteHeader(http.StatusNoContent)
				} else {
					http.Error(w, "origin not allowed", http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

package middleware

import (
	"net/http"
	"os"
	"strings"
)

func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get environment to determine CORS policy
			environment := os.Getenv("ENVIRONMENT")
			if environment == "" {
				environment = "development"
			}

			// In production, restrict to specific frontend domain
			// In development, allow all origins for easier local development
			if environment == "production" {
				// Get allowed frontend URL from environment variable
				allowedOrigin := os.Getenv("FRONTEND_URL")
				if allowedOrigin == "" {
					// Fallback to default production domain if not set
					allowedOrigin = "https://yourdomain.com"
				}

				origin := r.Header.Get("Origin")

				// Allow exact match or subdomain match
				isAllowed := origin == allowedOrigin
				if !isAllowed && strings.HasPrefix(origin, "https://") && strings.Contains(origin, allowedOrigin) {
					isAllowed = true
				}

				// Only set Access-Control-Allow-Origin if the origin matches
				if isAllowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				// If origin doesn't match, CORS headers are not set, blocking the request
			} else {
				// Development/staging: Allow requesting origin with credentials
				origin := r.Header.Get("Origin")
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					// Default to localhost if no origin header
					w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
				}
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions, Sec-WebSocket-Version, Sec-WebSocket-Key")
				w.WriteHeader(http.StatusOK)
				return // OPTIONS requests don't need to move past the CORS handler as their purpose is to negotiate access
			}

			next.ServeHTTP(w, r)
		})
	}
}

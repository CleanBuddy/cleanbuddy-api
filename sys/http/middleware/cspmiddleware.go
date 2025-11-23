package middleware

import (
	"net/http"
)

// CSPMiddleware returns a middleware that sets Content Security Policy and other security headers
func CSPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Content Security Policy
			// Restrictive policy for API endpoints
			csp := "default-src 'none'; " +
				"script-src 'none'; " +
				"style-src 'none'; " +
				"img-src 'none'; " +
				"font-src 'none'; " +
				"connect-src 'self'; " +
				"media-src 'none'; " +
				"object-src 'none'; " +
				"child-src 'none'; " +
				"frame-src 'none'; " +
				"worker-src 'none'; " +
				"frame-ancestors 'none'; " +
				"form-action 'none'; " +
				"base-uri 'none'; " +
				"manifest-src 'none'; " +
				"upgrade-insecure-requests; " +
				"block-all-mixed-content"

			w.Header().Set("Content-Security-Policy", csp)

			// Additional security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Strict-Transport-Security (HSTS) - only set for HTTPS
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CSPMiddlewareForPlayground returns a more permissive CSP for GraphQL playground
func CSPMiddlewareForPlayground() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// More permissive CSP for GraphQL playground
			csp := "default-src 'self' 'unsafe-inline' 'unsafe-eval' https: data: blob:; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https: data: blob:; " +
				"style-src 'self' 'unsafe-inline' https: data: blob:; " +
				"img-src 'self' data: https: blob:; " +
				"font-src 'self' https: data: blob:; " +
				"connect-src 'self' ws: wss: https:; " +
				"media-src 'self' https: data: blob:; " +
				"object-src 'none'; " +
				"child-src 'self'; " +
				"frame-src 'self'; " +
				"worker-src 'self'; " +
				"frame-ancestors 'none'; " +
				"form-action 'self'; " +
				"base-uri 'self'; " +
				"manifest-src 'self'"

			w.Header().Set("Content-Security-Policy", csp)

			// Additional security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY") // More restrictive anti-clickjacking protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Strict-Transport-Security (HSTS) - only set for HTTPS
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}

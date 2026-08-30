package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds baseline browser hardening headers (CSP + framing).
// Same-origin SPA via nginx/vite; connect-src allows WS for consoles.
func SecurityHeaders(cspEnabled bool) gin.HandlerFunc {
	csp := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob: https:",
		"font-src 'self' data:",
		"connect-src 'self' ws: wss:",
		"worker-src 'self' blob:",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"object-src 'none'",
	}, "; ")

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if cspEnabled {
			c.Header("Content-Security-Policy", csp)
		}
		c.Next()
	}
}

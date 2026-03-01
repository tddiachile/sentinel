package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SECURITY: never log sensitive data (passwords, tokens, keys, authorization headers)

// RequestLogger is a Fiber middleware that emits a structured slog log entry for
// every HTTP request after the response is written.
//
// It replaces the built-in github.com/gofiber/fiber/v2/middleware/logger and adds:
//   - Correlation ID propagation via LocalRequestID (set by RequestID middleware).
//   - Authenticated user ID and application ID when present in Fiber locals.
//   - Adaptive log level: ERROR for 5xx, WARN for 4xx, INFO for others.
//   - DEBUG level for noisy health-check and Swagger routes.
//
// The logger is injected as a dependency; there is no package-level global.
func RequestLogger(log *slog.Logger) fiber.Handler {
	httpLogger := log.With("component", "http")

	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Execute the next handler in the chain.
		chainErr := c.Next()

		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		status := c.Response().StatusCode()
		path := c.Path()
		method := c.Method()

		// Determine the effective log level for this request.
		level := resolveLevel(status, path)

		// Skip emission entirely when the resolved level is below the logger's
		// configured minimum.  slog.Logger.Enabled avoids allocating the record.
		if !httpLogger.Enabled(c.Context(), level) {
			return chainErr
		}

		// Build the list of structured attributes.
		attrs := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latencyMs,
			"ip", clientIP(c),
		}

		// Request ID — always expected when RequestID middleware is present.
		if rid, ok := c.Locals(LocalRequestID).(string); ok && rid != "" {
			attrs = append(attrs, "request_id", rid)
		}

		// User ID — only present on authenticated endpoints.
		if uid, ok := c.Locals(LocalActorID).(string); ok && uid != "" {
			attrs = append(attrs, "user_id", uid)
		}

		// App ID — only present when X-App-Key was validated.
		if aid, ok := c.Locals(LocalAppID).(string); ok && aid != "" {
			attrs = append(attrs, "app_id", aid)
		}

		httpLogger.Log(c.Context(), level, "HTTP request", attrs...)

		return chainErr
	}
}

// resolveLevel returns the slog.Level that should be used for a given HTTP
// status code and request path.
//
//   - /health and /swagger* routes -> DEBUG (high-frequency, low-signal)
//   - status >= 500               -> ERROR
//   - status >= 400               -> WARN
//   - everything else             -> INFO
func resolveLevel(status int, path string) slog.Level {
	if path == "/health" || strings.HasPrefix(path, "/swagger") {
		return slog.LevelDebug
	}
	if status >= 500 {
		return slog.LevelError
	}
	if status >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

// clientIP extracts the real client IP, preferring X-Forwarded-For when set.
// This mirrors the logic in AuditContext to keep IP resolution consistent.
func clientIP(c *fiber.Ctx) string {
	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		return c.IP()
	}
	// X-Forwarded-For may be a comma-separated list; take the first entry.
	if idx := strings.Index(ip, ","); idx != -1 {
		return strings.TrimSpace(ip[:idx])
	}
	return strings.TrimSpace(ip)
}

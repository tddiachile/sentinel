package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID is a Fiber middleware that reads or generates a correlation ID for
// each incoming request.
//
// Behaviour:
//  1. Read the X-Request-ID header from the incoming request.
//  2. If absent, generate a new UUID v4.
//  3. Store the value in c.Locals(LocalRequestID) so downstream handlers can
//     include it in structured logs and audit events.
//  4. Echo the value back in the X-Request-ID response header so callers can
//     correlate their request with server-side logs.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Locals(LocalRequestID, requestID)
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/enunezf/sentinel/internal/service"
)

// RequirePermission returns a middleware that checks if the authenticated user has the required permission.
func RequirePermission(authzSvc *service.AuthzService, permCode string, log *slog.Logger) fiber.Handler {
	logger := log.With("component", "permission_middleware")
	return func(c *fiber.Ctx) error {
		requestID, _ := c.Locals(LocalRequestID).(string)

		claims := GetClaims(c)
		if claims == nil {
			return respondError(c, fiber.StatusUnauthorized, "TOKEN_INVALID", "missing authentication")
		}

		allowed, err := authzSvc.HasPermission(c.Context(), claims, permCode)
		if err != nil {
			logger.Error("permission check failed",
				"error", err,
				"user_id", claims.Sub,
				"permission_code", permCode,
				"request_id", requestID,
			)
			return respondError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "authorization check failed")
		}
		if !allowed {
			logger.Info("permission denied",
				"user_id", claims.Sub,
				"permission_code", permCode,
				"request_id", requestID,
			)
			return respondError(c, fiber.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		}

		logger.Debug("permission granted",
			"user_id", claims.Sub,
			"permission_code", permCode,
			"request_id", requestID,
		)

		return c.Next()
	}
}

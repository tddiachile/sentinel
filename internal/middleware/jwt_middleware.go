package middleware

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/enunezf/sentinel/internal/domain"
	"github.com/enunezf/sentinel/internal/token"
)

// JWTAuth validates the Bearer token in the Authorization header using RS256.
func JWTAuth(tokenMgr *token.Manager, log *slog.Logger) fiber.Handler {
	logger := log.With("component", "jwt_middleware")
	return func(c *fiber.Ctx) error {
		requestID, _ := c.Locals(LocalRequestID).(string)

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			logger.Warn("missing Authorization header",
				"ip", c.IP(),
				"request_id", requestID,
			)
			return respondError(c, fiber.StatusUnauthorized, "TOKEN_INVALID", "missing Authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			logger.Warn("invalid Authorization header format",
				"ip", c.IP(),
				"request_id", requestID,
			)
			return respondError(c, fiber.StatusUnauthorized, "TOKEN_INVALID", "invalid Authorization header format")
		}

		tokenStr := parts[1]
		claims, err := tokenMgr.ValidateToken(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				logger.Warn("expired JWT token",
					"ip", c.IP(),
					"request_id", requestID,
				)
				return respondError(c, fiber.StatusUnauthorized, "TOKEN_EXPIRED", "access token has expired")
			}
			logger.Warn("invalid JWT token",
				"ip", c.IP(),
				"request_id", requestID,
			)
			return respondError(c, fiber.StatusUnauthorized, "TOKEN_INVALID", "invalid access token")
		}

		logger.Debug("JWT token validated",
			"request_id", requestID,
		)

		c.Locals(LocalClaims, claims)
		c.Locals(LocalActorID, claims.Sub)
		return c.Next()
	}
}

// GetClaims extracts JWT claims from fiber locals.
func GetClaims(c *fiber.Ctx) *domain.Claims {
	if v := c.Locals(LocalClaims); v != nil {
		if claims, ok := v.(*domain.Claims); ok {
			return claims
		}
	}
	return nil
}

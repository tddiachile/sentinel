package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserContext is the cached permissions context for a user (keyed by JWT jti).
type UserContext struct {
	UserID      string   `json:"user_id"`
	Application string   `json:"application"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	CostCenters []string `json:"cost_centers"`
}

// AuthzCache manages authorization caches in Redis.
type AuthzCache struct {
	client *redis.Client
	logger *slog.Logger
}

// NewAuthzCache creates a new AuthzCache.
func NewAuthzCache(client *redis.Client, log *slog.Logger) *AuthzCache {
	return &AuthzCache{
		client: client,
		logger: log.With("component", "authz_cache"),
	}
}

func userContextKey(jti string) string {
	return "user_context:" + jti
}

func permissionsMapKey(appSlug string) string {
	return "permissions_map:" + appSlug
}

func permissionsMapVersionKey(appSlug string) string {
	return "permissions_map_version:" + appSlug
}

// SetPermissions stores a user context (permissions) keyed by JWT jti.
func (c *AuthzCache) SetPermissions(ctx context.Context, jti string, uc *UserContext, ttl time.Duration) error {
	b, err := json.Marshal(uc)
	if err != nil {
		return fmt.Errorf("authz_cache: marshal user context: %w", err)
	}
	if err := c.client.Set(ctx, userContextKey(jti), string(b), ttl).Err(); err != nil {
		return fmt.Errorf("authz_cache: set permissions: %w", err)
	}
	return nil
}

// GetPermissions retrieves a user context by JWT jti. Returns nil if not found.
func (c *AuthzCache) GetPermissions(ctx context.Context, jti string) (*UserContext, error) {
	val, err := c.client.Get(ctx, userContextKey(jti)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("authz_cache: get permissions: %w", err)
	}
	var uc UserContext
	if err := json.Unmarshal([]byte(val), &uc); err != nil {
		return nil, fmt.Errorf("authz_cache: unmarshal user context: %w", err)
	}
	return &uc, nil
}

// DeletePermissions removes the cached user context for a jti.
func (c *AuthzCache) DeletePermissions(ctx context.Context, jti string) error {
	return c.client.Del(ctx, userContextKey(jti)).Err()
}

// SetPermissionsMap stores the full permissions map JSON for an app slug.
func (c *AuthzCache) SetPermissionsMap(ctx context.Context, appSlug string, mapData []byte, ttl time.Duration) error {
	if err := c.client.Set(ctx, permissionsMapKey(appSlug), string(mapData), ttl).Err(); err != nil {
		return fmt.Errorf("authz_cache: set permissions map: %w", err)
	}
	return nil
}

// GetPermissionsMap retrieves the permissions map JSON for an app slug.
func (c *AuthzCache) GetPermissionsMap(ctx context.Context, appSlug string) ([]byte, error) {
	val, err := c.client.Get(ctx, permissionsMapKey(appSlug)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("authz_cache: get permissions map: %w", err)
	}
	return []byte(val), nil
}

// SetPermissionsMapVersion stores the current version hash for an app's permissions map.
func (c *AuthzCache) SetPermissionsMapVersion(ctx context.Context, appSlug, version string, ttl time.Duration) error {
	if err := c.client.Set(ctx, permissionsMapVersionKey(appSlug), version, ttl).Err(); err != nil {
		return fmt.Errorf("authz_cache: set permissions map version: %w", err)
	}
	return nil
}

// GetPermissionsMapVersion retrieves the current version hash for an app's permissions map.
func (c *AuthzCache) GetPermissionsMapVersion(ctx context.Context, appSlug string) (string, error) {
	val, err := c.client.Get(ctx, permissionsMapVersionKey(appSlug)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("authz_cache: get permissions map version: %w", err)
	}
	return val, nil
}

// InvalidatePermissionsMap removes the cached permissions map and version for an app.
func (c *AuthzCache) InvalidatePermissionsMap(ctx context.Context, appSlug string) error {
	c.client.Del(ctx, permissionsMapKey(appSlug))
	c.client.Del(ctx, permissionsMapVersionKey(appSlug))
	return nil
}

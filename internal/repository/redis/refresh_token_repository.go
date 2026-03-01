package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RefreshTokenData is the value stored in Redis for each refresh token.
// TokenHash stores the bcrypt hash for PG lookup. Redis key is the raw token (UUID v4).
type RefreshTokenData struct {
	UserID     string `json:"user_id"`
	AppID      string `json:"app_id"`
	ExpiresAt  string `json:"expires_at"`
	ClientType string `json:"client_type"`
	UserAgent  string `json:"user_agent"`
	IP         string `json:"ip"`
	TokenHash  string `json:"token_hash"` // bcrypt hash, used to look up PG record
}

// RefreshTokenRepository manages refresh tokens in Redis.
type RefreshTokenRepository struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository.
func NewRefreshTokenRepository(client *redis.Client, log *slog.Logger) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		client: client,
		logger: log.With("component", "redis_refresh_repo"),
	}
}

func refreshKey(hash string) string {
	return "refresh:" + hash
}

// Set stores refresh token data in Redis with a TTL.
func (r *RefreshTokenRepository) Set(ctx context.Context, hash string, data RefreshTokenData, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("redis_refresh: marshal: %w", err)
	}
	if err := r.client.Set(ctx, refreshKey(hash), string(b), ttl).Err(); err != nil {
		return fmt.Errorf("redis_refresh: set: %w", err)
	}
	return nil
}

// Get retrieves refresh token data from Redis by hash. Returns nil if not found.
func (r *RefreshTokenRepository) Get(ctx context.Context, hash string) (*RefreshTokenData, error) {
	val, err := r.client.Get(ctx, refreshKey(hash)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis_refresh: get: %w", err)
	}
	var data RefreshTokenData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("redis_refresh: unmarshal: %w", err)
	}
	return &data, nil
}

// Delete removes a refresh token from Redis by hash.
func (r *RefreshTokenRepository) Delete(ctx context.Context, hash string) error {
	if err := r.client.Del(ctx, refreshKey(hash)).Err(); err != nil {
		return fmt.Errorf("redis_refresh: delete: %w", err)
	}
	return nil
}

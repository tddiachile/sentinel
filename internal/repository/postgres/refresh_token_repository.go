package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/enunezf/sentinel/internal/domain"
)

// RefreshTokenRepository handles persistence of RefreshToken entities.
type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository.
func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create inserts a new refresh token record.
func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	deviceInfoJSON, err := json.Marshal(token.DeviceInfo)
	if err != nil {
		return fmt.Errorf("refresh_token_repo: marshal device_info: %w", err)
	}

	const q = `
		INSERT INTO refresh_tokens (id, user_id, app_id, token_hash, device_info, expires_at, is_revoked, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, FALSE, NOW())`
	_, err = r.db.Exec(ctx, q,
		token.ID, token.UserID, token.AppID, token.TokenHash,
		deviceInfoJSON, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("refresh_token_repo: create: %w", err)
	}
	return nil
}

// FindByHash returns the refresh token matching the given hash, or nil.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	const q = `
		SELECT id, user_id, app_id, token_hash, device_info, expires_at, used_at, is_revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	row := r.db.QueryRow(ctx, q, hash)

	var t domain.RefreshToken
	var deviceInfoRaw []byte
	err := row.Scan(
		&t.ID, &t.UserID, &t.AppID, &t.TokenHash,
		&deviceInfoRaw, &t.ExpiresAt, &t.UsedAt, &t.IsRevoked, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("refresh_token_repo: find by hash: %w", err)
	}
	if len(deviceInfoRaw) > 0 {
		_ = json.Unmarshal(deviceInfoRaw, &t.DeviceInfo)
	}
	return &t, nil
}

// Revoke marks a refresh token as revoked by ID.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET is_revoked = TRUE WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("refresh_token_repo: revoke: %w", err)
	}
	return nil
}

// RevokeAllForUser marks all refresh tokens for a user+app as revoked.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID, appID uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET is_revoked = TRUE WHERE user_id = $1 AND app_id = $2 AND is_revoked = FALSE`
	_, err := r.db.Exec(ctx, q, userID, appID)
	if err != nil {
		return fmt.Errorf("refresh_token_repo: revoke all for user: %w", err)
	}
	return nil
}

// FindByRawToken scans non-revoked tokens for a user and compares bcrypt hashes.
// This is the fallback when Redis is unavailable. O(n) in number of active tokens.
func (r *RefreshTokenRepository) FindByRawToken(ctx context.Context, rawToken string) (*domain.RefreshToken, error) {
	const q = `
		SELECT id, user_id, app_id, token_hash, device_info, expires_at, used_at, is_revoked, created_at
		FROM refresh_tokens
		WHERE is_revoked = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1000`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("refresh_token_repo: find by raw token scan: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t domain.RefreshToken
		var deviceInfoRaw []byte
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AppID, &t.TokenHash,
			&deviceInfoRaw, &t.ExpiresAt, &t.UsedAt, &t.IsRevoked, &t.CreatedAt,
		); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(rawToken)) == nil {
			if len(deviceInfoRaw) > 0 {
				_ = json.Unmarshal(deviceInfoRaw, &t.DeviceInfo)
			}
			return &t, nil
		}
	}
	return nil, rows.Err()
}

// RevokeAllForUserAllApps marks all refresh tokens for a user (all apps) as revoked.
func (r *RefreshTokenRepository) RevokeAllForUserAllApps(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET is_revoked = TRUE WHERE user_id = $1 AND is_revoked = FALSE`
	_, err := r.db.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("refresh_token_repo: revoke all for user all apps: %w", err)
	}
	return nil
}

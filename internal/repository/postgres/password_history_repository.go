package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PasswordHistoryRepository manages the password_history table.
type PasswordHistoryRepository struct {
	db *pgxpool.Pool
}

// NewPasswordHistoryRepository creates a new PasswordHistoryRepository.
func NewPasswordHistoryRepository(db *pgxpool.Pool) *PasswordHistoryRepository {
	return &PasswordHistoryRepository{db: db}
}

// GetLastN returns the last n password hashes for the user, ordered by created_at DESC.
func (r *PasswordHistoryRepository) GetLastN(ctx context.Context, userID uuid.UUID, n int) ([]string, error) {
	const q = `
		SELECT password_hash
		FROM password_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, userID, n)
	if err != nil {
		return nil, fmt.Errorf("password_history: get last n: %w", err)
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("password_history: scan: %w", err)
		}
		hashes = append(hashes, h)
	}
	return hashes, rows.Err()
}

// Add inserts a password hash into the history for a user.
func (r *PasswordHistoryRepository) Add(ctx context.Context, userID uuid.UUID, hash string) error {
	const q = `
		INSERT INTO password_history (id, user_id, password_hash, created_at)
		VALUES (gen_random_uuid(), $1, $2, NOW())`

	if _, err := r.db.Exec(ctx, q, userID, hash); err != nil {
		return fmt.Errorf("password_history: add: %w", err)
	}
	return nil
}

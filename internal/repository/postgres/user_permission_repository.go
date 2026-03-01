package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/enunezf/sentinel/internal/domain"
)

// UserPermissionRepository manages user_permissions assignments.
type UserPermissionRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewUserPermissionRepository creates a new UserPermissionRepository.
func NewUserPermissionRepository(db *pgxpool.Pool, log *slog.Logger) *UserPermissionRepository {
	return &UserPermissionRepository{
		db:     db,
		logger: log.With("component", "user_perm_repo"),
	}
}

// Assign creates a new user_permission assignment.
func (r *UserPermissionRepository) Assign(ctx context.Context, up *domain.UserPermission) error {
	const q = `
		INSERT INTO user_permissions (id, user_id, permission_id, application_id, granted_by, valid_from, valid_until, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, NOW())`
	_, err := r.db.Exec(ctx, q,
		up.ID, up.UserID, up.PermissionID, up.ApplicationID,
		up.GrantedBy, up.ValidFrom, up.ValidUntil,
	)
	if err != nil {
		return fmt.Errorf("user_perm_repo: assign: %w", err)
	}
	return nil
}

// Revoke marks a user_permission assignment as inactive.
func (r *UserPermissionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE user_permissions SET is_active = FALSE WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("user_perm_repo: revoke: %w", err)
	}
	return nil
}

// FindByID returns a user_permission by ID, or nil.
func (r *UserPermissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.UserPermission, error) {
	const q = `
		SELECT up.id, up.user_id, up.permission_id, up.application_id, up.granted_by,
		       up.valid_from, up.valid_until, up.is_active, up.created_at, p.code
		FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id
		WHERE up.id = $1`
	row := r.db.QueryRow(ctx, q, id)
	var up domain.UserPermission
	err := row.Scan(&up.ID, &up.UserID, &up.PermissionID, &up.ApplicationID, &up.GrantedBy,
		&up.ValidFrom, &up.ValidUntil, &up.IsActive, &up.CreatedAt, &up.PermissionCode)
	if err != nil {
		return nil, fmt.Errorf("user_perm_repo: find by id: %w", err)
	}
	return &up, nil
}

// ListForUser returns all user_permissions for a user (all apps).
func (r *UserPermissionRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserPermission, error) {
	const q = `
		SELECT up.id, up.user_id, up.permission_id, up.application_id, up.granted_by,
		       up.valid_from, up.valid_until, up.is_active, up.created_at, p.code
		FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id
		WHERE up.user_id = $1
		ORDER BY up.created_at DESC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("user_perm_repo: list for user: %w", err)
	}
	defer rows.Close()

	var result []*domain.UserPermission
	for rows.Next() {
		var up domain.UserPermission
		if err := rows.Scan(&up.ID, &up.UserID, &up.PermissionID, &up.ApplicationID, &up.GrantedBy,
			&up.ValidFrom, &up.ValidUntil, &up.IsActive, &up.CreatedAt, &up.PermissionCode); err != nil {
			return nil, fmt.Errorf("user_perm_repo: scan: %w", err)
		}
		result = append(result, &up)
	}
	return result, rows.Err()
}

// GetActivePermissionCodesForUserApp returns active permission codes for a user+app.
func (r *UserPermissionRepository) GetActivePermissionCodesForUserApp(ctx context.Context, userID, appID uuid.UUID) ([]string, error) {
	const q = `
		SELECT DISTINCT p.code
		FROM user_permissions up
		JOIN permissions p ON p.id = up.permission_id
		WHERE up.user_id = $1
		  AND up.application_id = $2
		  AND up.is_active = TRUE
		  AND up.valid_from <= NOW()
		  AND (up.valid_until IS NULL OR up.valid_until > NOW())`
	rows, err := r.db.Query(ctx, q, userID, appID)
	if err != nil {
		return nil, fmt.Errorf("user_perm_repo: get active codes: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("user_perm_repo: scan code: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

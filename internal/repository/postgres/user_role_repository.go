package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/enunezf/sentinel/internal/domain"
)

// UserRoleRepository manages user_roles assignments.
type UserRoleRepository struct {
	db *pgxpool.Pool
}

// NewUserRoleRepository creates a new UserRoleRepository.
func NewUserRoleRepository(db *pgxpool.Pool) *UserRoleRepository {
	return &UserRoleRepository{db: db}
}

// Assign creates a new user_role assignment.
func (r *UserRoleRepository) Assign(ctx context.Context, ur *domain.UserRole) error {
	const q = `
		INSERT INTO user_roles (id, user_id, role_id, application_id, granted_by, valid_from, valid_until, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, NOW())`
	_, err := r.db.Exec(ctx, q,
		ur.ID, ur.UserID, ur.RoleID, ur.ApplicationID,
		ur.GrantedBy, ur.ValidFrom, ur.ValidUntil,
	)
	if err != nil {
		return fmt.Errorf("user_role_repo: assign: %w", err)
	}
	return nil
}

// Revoke marks a user_role assignment as inactive.
func (r *UserRoleRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE user_roles SET is_active = FALSE WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("user_role_repo: revoke: %w", err)
	}
	return nil
}

// FindByID returns the user_role by ID, or nil.
func (r *UserRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.UserRole, error) {
	const q = `
		SELECT ur.id, ur.user_id, ur.role_id, ur.application_id, ur.granted_by,
		       ur.valid_from, ur.valid_until, ur.is_active, ur.created_at, r.name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.id = $1`
	row := r.db.QueryRow(ctx, q, id)
	var ur domain.UserRole
	err := row.Scan(&ur.ID, &ur.UserID, &ur.RoleID, &ur.ApplicationID, &ur.GrantedBy,
		&ur.ValidFrom, &ur.ValidUntil, &ur.IsActive, &ur.CreatedAt, &ur.RoleName)
	if err != nil {
		return nil, fmt.Errorf("user_role_repo: find by id: %w", err)
	}
	return &ur, nil
}

// ListForUser returns all user_roles for a user (all apps).
func (r *UserRoleRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserRole, error) {
	const q = `
		SELECT ur.id, ur.user_id, ur.role_id, ur.application_id, ur.granted_by,
		       ur.valid_from, ur.valid_until, ur.is_active, ur.created_at, r.name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY ur.created_at DESC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("user_role_repo: list for user: %w", err)
	}
	defer rows.Close()

	var result []*domain.UserRole
	for rows.Next() {
		var ur domain.UserRole
		if err := rows.Scan(&ur.ID, &ur.UserID, &ur.RoleID, &ur.ApplicationID, &ur.GrantedBy,
			&ur.ValidFrom, &ur.ValidUntil, &ur.IsActive, &ur.CreatedAt, &ur.RoleName); err != nil {
			return nil, fmt.Errorf("user_role_repo: scan: %w", err)
		}
		result = append(result, &ur)
	}
	return result, rows.Err()
}

// GetActiveRoleNamesForUserApp returns active role names for a user+app (used for JWT).
func (r *UserRoleRepository) GetActiveRoleNamesForUserApp(ctx context.Context, userID, appID uuid.UUID) ([]string, error) {
	const q = `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		  AND ur.application_id = $2
		  AND ur.is_active = TRUE
		  AND r.is_active = TRUE
		  AND ur.valid_from <= NOW()
		  AND (ur.valid_until IS NULL OR ur.valid_until > NOW())`
	rows, err := r.db.Query(ctx, q, userID, appID)
	if err != nil {
		return nil, fmt.Errorf("user_role_repo: get active role names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("user_role_repo: scan role name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

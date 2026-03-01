package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/enunezf/sentinel/internal/domain"
)

// RoleRepository handles persistence of Role entities.
type RoleRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewRoleRepository creates a new RoleRepository.
func NewRoleRepository(db *pgxpool.Pool, log *slog.Logger) *RoleRepository {
	return &RoleRepository{
		db:     db,
		logger: log.With("component", "role_repo"),
	}
}

func scanRole(row pgx.Row) (*domain.Role, error) {
	var r domain.Role
	err := row.Scan(&r.ID, &r.ApplicationID, &r.Name, &r.Description, &r.IsSystem, &r.IsActive, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// FindByID returns the role matching id, or nil.
func (r *RoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	const q = `SELECT id, application_id, name, description, is_system, is_active, created_at, updated_at FROM roles WHERE id = $1`
	row := r.db.QueryRow(ctx, q, id)
	role, err := scanRole(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("role_repo: find by id: %w", err)
	}
	return role, nil
}

// FindByNameAndApp returns the role matching name+appID, or nil.
func (r *RoleRepository) FindByNameAndApp(ctx context.Context, name string, appID uuid.UUID) (*domain.Role, error) {
	const q = `SELECT id, application_id, name, description, is_system, is_active, created_at, updated_at FROM roles WHERE name = $1 AND application_id = $2`
	row := r.db.QueryRow(ctx, q, name, appID)
	role, err := scanRole(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("role_repo: find by name and app: %w", err)
	}
	return role, nil
}

// Create inserts a new role.
func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	const q = `
		INSERT INTO roles (id, application_id, name, description, is_system, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`
	_, err := r.db.Exec(ctx, q, role.ID, role.ApplicationID, role.Name, role.Description, role.IsSystem, role.IsActive)
	if err != nil {
		return fmt.Errorf("role_repo: create: %w", err)
	}
	return nil
}

// Update updates name and description of a role.
func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) error {
	const q = `
		UPDATE roles SET name = $2, description = $3, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, role.ID, role.Name, role.Description)
	if err != nil {
		return fmt.Errorf("role_repo: update: %w", err)
	}
	return nil
}

// Deactivate sets is_active = false for a role.
func (r *RoleRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE roles SET is_active = FALSE, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("role_repo: deactivate: %w", err)
	}
	return nil
}

// RoleFilter defines filters for listing roles.
type RoleFilter struct {
	ApplicationID *uuid.UUID
	Page          int
	PageSize      int
}

// List returns a paginated list of roles and total count.
func (r *RoleRepository) List(ctx context.Context, filter RoleFilter) ([]*domain.Role, int, error) {
	args := []interface{}{}
	where := ""
	if filter.ApplicationID != nil {
		where = "WHERE application_id = $1"
		args = append(args, *filter.ApplicationID)
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM roles `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("role_repo: list count: %w", err)
	}

	idx := len(args) + 1
	dataQ := `SELECT id, application_id, name, description, is_system, is_active, created_at, updated_at FROM roles ` +
		where + fmt.Sprintf(` ORDER BY name ASC LIMIT $%d OFFSET $%d`, idx, idx+1)
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("role_repo: list query: %w", err)
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("role_repo: scan: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, total, rows.Err()
}

// GetPermissionsCount returns the count of permissions assigned to a role.
func (r *RoleRepository) GetPermissionsCount(ctx context.Context, roleID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM role_permissions WHERE role_id = $1`, roleID).Scan(&count)
	return count, err
}

// GetUsersCount returns the count of users assigned to a role.
func (r *RoleRepository) GetUsersCount(ctx context.Context, roleID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM user_roles WHERE role_id = $1 AND is_active = TRUE`, roleID).Scan(&count)
	return count, err
}

// GetPermissions returns all permissions for a role.
func (r *RoleRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	const q = `
		SELECT p.id, p.application_id, p.code, COALESCE(p.description,''), p.scope_type, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1`
	rows, err := r.db.Query(ctx, q, roleID)
	if err != nil {
		return nil, fmt.Errorf("role_repo: get permissions: %w", err)
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.ApplicationID, &p.Code, &p.Description, &p.ScopeType, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("role_repo: scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// AddPermission adds a permission to a role.
func (r *RoleRepository) AddPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	const q = `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, q, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("role_repo: add permission: %w", err)
	}
	return nil
}

// RemovePermission removes a permission from a role.
func (r *RoleRepository) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	const q = `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.Exec(ctx, q, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("role_repo: remove permission: %w", err)
	}
	return nil
}

// GetRolesForPermission returns names of all active roles that have a given permission.
func (r *RoleRepository) GetRolesForPermission(ctx context.Context, permissionID uuid.UUID) ([]string, error) {
	const q = `
		SELECT r.name
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		WHERE rp.permission_id = $1 AND r.is_active = TRUE`
	rows, err := r.db.Query(ctx, q, permissionID)
	if err != nil {
		return nil, fmt.Errorf("role_repo: get roles for permission: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("role_repo: scan role name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a named collection of permissions for an application.
type Role struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	Name          string
	Description   string
	IsSystem      bool
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserRole represents the assignment of a role to a user for an application.
type UserRole struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	RoleID        uuid.UUID
	ApplicationID uuid.UUID
	GrantedBy     uuid.UUID
	ValidFrom     time.Time
	ValidUntil    *time.Time
	IsActive      bool
	CreatedAt     time.Time
	// Populated via join.
	RoleName string
}

// RoleWithPermissions extends Role with its permissions list.
type RoleWithPermissions struct {
	Role
	Permissions []Permission
	UsersCount  int
}

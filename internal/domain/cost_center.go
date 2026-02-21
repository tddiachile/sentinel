package domain

import (
	"time"

	"github.com/google/uuid"
)

// CostCenter represents a cost center (centro de costo) entity.
type CostCenter struct {
	ID            uuid.UUID
	ApplicationID uuid.UUID
	Code          string
	Name          string
	IsActive      bool
	CreatedAt     time.Time
}

// UserCostCenter represents the assignment of a cost center to a user.
type UserCostCenter struct {
	UserID        uuid.UUID
	CostCenterID  uuid.UUID
	ApplicationID uuid.UUID
	GrantedBy     uuid.UUID
	ValidFrom     time.Time
	ValidUntil    *time.Time
	// Populated via join.
	CostCenterCode string
	CostCenterName string
}

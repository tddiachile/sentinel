package domain

import (
	"time"

	"github.com/google/uuid"
)

// Application represents a registered client application (tenant).
type Application struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	SecretKey string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

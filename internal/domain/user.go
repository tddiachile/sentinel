package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a Sentinel user account.
type User struct {
	ID             uuid.UUID
	Username       string
	Email          string
	PasswordHash   string
	IsActive       bool
	MustChangePwd  bool
	LastLoginAt    *time.Time
	FailedAttempts int
	LockedUntil    *time.Time
	LockoutCount   int
	LockoutDate    *time.Time // stored as DATE in PG, mapped to time.Time truncated to day
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsLocked returns true if the user account is currently locked.
// A nil LockedUntil with LockoutCount >= 3 means permanent lock.
func (u *User) IsLocked(now time.Time) bool {
	if u.LockedUntil == nil {
		// Permanent lock: lockout_count >= 3 and locked_until IS NULL (was set that way)
		// We detect permanent lock by checking if LockedUntil is nil AND lockoutCount >= 3
		// AND the user was actually locked (not just never locked before).
		// We use LockoutDate as the signal that a lock happened.
		return u.LockoutCount >= 3 && u.LockoutDate != nil
	}
	return now.Before(*u.LockedUntil)
}

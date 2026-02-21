package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of an audit event.
type EventType string

// Authentication events.
const (
	EventAuthLoginSuccess    EventType = "AUTH_LOGIN_SUCCESS"
	EventAuthLoginFailed     EventType = "AUTH_LOGIN_FAILED"
	EventAuthLogout          EventType = "AUTH_LOGOUT"
	EventAuthTokenRefreshed  EventType = "AUTH_TOKEN_REFRESHED"
	EventAuthPasswordChanged EventType = "AUTH_PASSWORD_CHANGED"
	EventAuthPasswordReset   EventType = "AUTH_PASSWORD_RESET"
	EventAuthAccountLocked   EventType = "AUTH_ACCOUNT_LOCKED"
)

// Authorization events.
const (
	EventAuthzPermissionGranted EventType = "AUTHZ_PERMISSION_GRANTED"
	EventAuthzPermissionDenied  EventType = "AUTHZ_PERMISSION_DENIED"
)

// User management events.
const (
	EventUserCreated     EventType = "USER_CREATED"
	EventUserUpdated     EventType = "USER_UPDATED"
	EventUserDeactivated EventType = "USER_DEACTIVATED"
	EventUserUnlocked    EventType = "USER_UNLOCKED"
)

// Role management events.
const (
	EventRoleCreated            EventType = "ROLE_CREATED"
	EventRoleUpdated            EventType = "ROLE_UPDATED"
	EventRoleDeleted            EventType = "ROLE_DELETED"
	EventRolePermissionAssigned EventType = "ROLE_PERMISSION_ASSIGNED"
	EventRolePermissionRevoked  EventType = "ROLE_PERMISSION_REVOKED"
)

// Assignment events.
const (
	EventUserRoleAssigned       EventType = "USER_ROLE_ASSIGNED"
	EventUserRoleRevoked        EventType = "USER_ROLE_REVOKED"
	EventUserPermissionAssigned EventType = "USER_PERMISSION_ASSIGNED"
	EventUserPermissionRevoked  EventType = "USER_PERMISSION_REVOKED"
	EventUserCostCenterAssigned EventType = "USER_COST_CENTER_ASSIGNED"
)

// System events.
const (
	EventSystemBootstrap EventType = "SYSTEM_BOOTSTRAP"
)

// AuditLog represents a single immutable audit record.
type AuditLog struct {
	ID            uuid.UUID
	EventType     EventType
	ApplicationID *uuid.UUID
	UserID        *uuid.UUID
	ActorID       *uuid.UUID
	ResourceType  *string
	ResourceID    *uuid.UUID
	OldValue      map[string]interface{}
	NewValue      map[string]interface{}
	IPAddress     string
	UserAgent     string
	Success       bool
	ErrorMessage  string
	CreatedAt     time.Time
}

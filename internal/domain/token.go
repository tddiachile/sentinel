package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token stored in PostgreSQL.
type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	AppID      uuid.UUID
	TokenHash  string
	DeviceInfo DeviceInfo
	ExpiresAt  time.Time
	UsedAt     *time.Time
	IsRevoked  bool
	CreatedAt  time.Time
}

// DeviceInfo holds the client context stored in refresh_tokens.device_info (JSONB).
type DeviceInfo struct {
	UserAgent  string `json:"user_agent"`
	IP         string `json:"ip"`
	ClientType string `json:"client_type"`
}

// ClientType represents the enum of allowed client types.
type ClientType string

const (
	ClientTypeWeb     ClientType = "web"
	ClientTypeMobile  ClientType = "mobile"
	ClientTypeDesktop ClientType = "desktop"
)

// IsValidClientType returns true if ct is one of the allowed client types.
func IsValidClientType(ct string) bool {
	switch ClientType(ct) {
	case ClientTypeWeb, ClientTypeMobile, ClientTypeDesktop:
		return true
	}
	return false
}

// Claims represents the JWT payload claims for Sentinel access tokens.
type Claims struct {
	Sub      string   `json:"sub"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	App      string   `json:"app"`
	Roles    []string `json:"roles"`
	Iat      int64    `json:"iat"`
	Exp      int64    `json:"exp"`
	Jti      string   `json:"jti"`
}

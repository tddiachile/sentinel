package service

// AuthServiceI is a testable variant of AuthService that accepts interfaces
// instead of concrete repository types. It is functionally equivalent to
// AuthService and is used in unit tests to enable full mock injection.

import (
	"context"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"

	"github.com/enunezf/sentinel/internal/config"
	"github.com/enunezf/sentinel/internal/domain"
	redisrepo "github.com/enunezf/sentinel/internal/repository/redis"
	"github.com/enunezf/sentinel/internal/token"
)

// AuthServiceI wraps the auth business logic using interface-based dependencies.
type AuthServiceI struct {
	userRepo         UserRepositoryIface
	appRepo          ApplicationRepositoryIface
	refreshPGRepo    RefreshTokenPGRepositoryIface
	refreshRedisRepo RefreshTokenRedisRepositoryIface
	pwdHistoryRepo   PasswordHistoryRepositoryIface
	userRoleRepo     UserRoleRepositoryIface
	tokenMgr         *token.Manager
	auditSvc         AuditServiceIface
	cfg              *config.Config
}

// NewAuthServiceI creates an AuthServiceI with interface-based dependencies.
func NewAuthServiceI(
	userRepo UserRepositoryIface,
	appRepo ApplicationRepositoryIface,
	refreshPGRepo RefreshTokenPGRepositoryIface,
	refreshRedisRepo RefreshTokenRedisRepositoryIface,
	pwdHistoryRepo PasswordHistoryRepositoryIface,
	userRoleRepo UserRoleRepositoryIface,
	tokenMgr *token.Manager,
	auditSvc AuditServiceIface,
	cfg *config.Config,
) *AuthServiceI {
	return &AuthServiceI{
		userRepo:         userRepo,
		appRepo:          appRepo,
		refreshPGRepo:    refreshPGRepo,
		refreshRedisRepo: refreshRedisRepo,
		pwdHistoryRepo:   pwdHistoryRepo,
		userRoleRepo:     userRoleRepo,
		tokenMgr:         tokenMgr,
		auditSvc:         auditSvc,
		cfg:              cfg,
	}
}

// Login validates credentials and returns tokens on success.
func (s *AuthServiceI) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	app, err := s.appRepo.FindBySecretKey(ctx, req.AppKey)
	if err != nil {
		return nil, fmt.Errorf("auth: find app: %w", err)
	}
	if app == nil || !app.IsActive {
		return nil, ErrApplicationNotFound
	}

	if !domain.IsValidClientType(req.ClientType) {
		return nil, ErrInvalidClientType
	}

	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("auth: find user: %w", err)
	}

	if user == nil {
		appID := app.ID
		s.auditSvc.LogEvent(&domain.AuditLog{
			EventType:     domain.EventAuthLoginFailed,
			ApplicationID: &appID,
			IPAddress:     req.IP,
			UserAgent:     req.UserAgent,
			Success:       false,
			ErrorMessage:  "user not found",
		})
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	now := time.Now()
	if user.IsLocked(now) {
		return nil, ErrAccountLocked
	}

	normalizedPwd := norm.NFC.String(req.Password)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(normalizedPwd)); err != nil {
		s.handleIFailedLogin(ctx, user, app.ID, req.IP, req.UserAgent)
		return nil, ErrInvalidCredentials
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("auth: update last login: %w", err)
	}

	refreshTTL := s.cfg.JWT.RefreshTokenTTLWeb
	if req.ClientType == string(domain.ClientTypeMobile) || req.ClientType == string(domain.ClientTypeDesktop) {
		refreshTTL = s.cfg.JWT.RefreshTokenTTLMobile
	}

	roles, err := s.userRoleRepo.GetActiveRoleNamesForUserApp(ctx, user.ID, app.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: get roles: %w", err)
	}

	accessToken, err := s.tokenMgr.GenerateAccessToken(user, app.Slug, roles, s.cfg.JWT.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: generate access token: %w", err)
	}

	rawRefreshToken := uuid.New().String()
	if err := s.istoreRefreshToken(ctx, user.ID, app.ID, rawRefreshToken, req.ClientType, req.IP, req.UserAgent, refreshTTL); err != nil {
		return nil, fmt.Errorf("auth: store refresh token: %w", err)
	}

	userID := user.ID
	appID := app.ID
	resType := "user"
	s.auditSvc.LogEvent(&domain.AuditLog{
		EventType:     domain.EventAuthLoginSuccess,
		ApplicationID: &appID,
		UserID:        &userID,
		ActorID:       &userID,
		ResourceType:  &resType,
		ResourceID:    &userID,
		IPAddress:     req.IP,
		UserAgent:     req.UserAgent,
		Success:       true,
	})

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWT.AccessTokenTTL.Seconds()),
		User:         user,
	}, nil
}

func (s *AuthServiceI) handleIFailedLogin(ctx context.Context, user *domain.User, appID uuid.UUID, ip, ua string) {
	user.FailedAttempts++
	var lockedUntil *time.Time
	lockoutCount := user.LockoutCount
	lockoutDate := user.LockoutDate
	now := time.Now()

	if user.FailedAttempts >= s.cfg.Security.MaxFailedAttempts {
		today := now.UTC().Truncate(24 * time.Hour)
		if lockoutDate == nil || !lockoutDate.UTC().Truncate(24*time.Hour).Equal(today) {
			lockoutCount = 0
			lockoutDate = &today
		}
		lockoutCount++

		if lockoutCount >= 3 {
			lockedUntil = nil
		} else {
			t := now.Add(s.cfg.Security.LockoutDuration)
			lockedUntil = &t
		}

		userID := user.ID
		resType := "user"
		s.auditSvc.LogEvent(&domain.AuditLog{
			EventType:     domain.EventAuthAccountLocked,
			ApplicationID: &appID,
			UserID:        &userID,
			ResourceType:  &resType,
			ResourceID:    &userID,
			IPAddress:     ip,
			UserAgent:     ua,
			Success:       true,
		})
	}

	_ = s.userRepo.UpdateFailedAttempts(ctx, user.ID, user.FailedAttempts, lockedUntil, lockoutCount, lockoutDate)

	userID := user.ID
	resType := "user"
	s.auditSvc.LogEvent(&domain.AuditLog{
		EventType:     domain.EventAuthLoginFailed,
		ApplicationID: &appID,
		UserID:        &userID,
		ResourceType:  &resType,
		ResourceID:    &userID,
		NewValue:      map[string]interface{}{"failed_attempts": user.FailedAttempts},
		IPAddress:     ip,
		UserAgent:     ua,
		Success:       false,
		ErrorMessage:  "Invalid credentials",
	})
}

func (s *AuthServiceI) istoreRefreshToken(ctx context.Context, userID, appID uuid.UUID, rawToken, clientType, ip, ua string, ttl time.Duration) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(rawToken), s.cfg.Security.BcryptCost)
	if err != nil {
		return fmt.Errorf("auth: hash refresh token: %w", err)
	}
	hashStr := string(hash)

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		AppID:     appID,
		TokenHash: hashStr,
		DeviceInfo: domain.DeviceInfo{
			UserAgent:  ua,
			IP:         ip,
			ClientType: clientType,
		},
		ExpiresAt: time.Now().Add(ttl),
	}

	if err := s.refreshPGRepo.Create(ctx, rt); err != nil {
		return fmt.Errorf("auth: pg create refresh token: %w", err)
	}

	redisData := redisrepo.RefreshTokenData{
		UserID:     userID.String(),
		AppID:      appID.String(),
		ExpiresAt:  rt.ExpiresAt.Format(time.RFC3339),
		ClientType: clientType,
		UserAgent:  ua,
		IP:         ip,
		TokenHash:  hashStr,
	}
	_ = s.refreshRedisRepo.Set(ctx, rawToken, redisData, ttl)
	return nil
}

func (s *AuthServiceI) ifindRefreshToken(ctx context.Context, rawToken string) (*domain.RefreshToken, error) {
	data, err := s.refreshRedisRepo.Get(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("auth: redis get refresh token: %w", err)
	}

	if data != nil && data.TokenHash != "" {
		rt, err := s.refreshPGRepo.FindByHash(ctx, data.TokenHash)
		if err != nil {
			return nil, fmt.Errorf("auth: pg find refresh by hash: %w", err)
		}
		return rt, nil
	}

	return s.refreshPGRepo.FindByRawToken(ctx, rawToken)
}

// Refresh validates a refresh token and issues new tokens (rotation).
func (s *AuthServiceI) Refresh(ctx context.Context, req RefreshRequest) (*RefreshResponse, error) {
	app, err := s.appRepo.FindBySecretKey(ctx, req.AppKey)
	if err != nil || app == nil || !app.IsActive {
		return nil, ErrApplicationNotFound
	}

	rtRecord, err := s.ifindRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("auth: find refresh token: %w", err)
	}
	if rtRecord == nil {
		return nil, ErrTokenInvalid
	}
	if rtRecord.IsRevoked {
		return nil, ErrTokenRevoked
	}
	if time.Now().After(rtRecord.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	user, err := s.userRepo.FindByID(ctx, rtRecord.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}
	if !user.IsActive {
		return nil, ErrAccountInactive
	}
	if user.IsLocked(time.Now()) {
		return nil, ErrAccountLocked
	}

	if err := s.refreshPGRepo.Revoke(ctx, rtRecord.ID); err != nil {
		return nil, fmt.Errorf("auth: revoke refresh token: %w", err)
	}
	_ = s.refreshRedisRepo.Delete(ctx, req.RefreshToken)

	roles, err := s.userRoleRepo.GetActiveRoleNamesForUserApp(ctx, user.ID, rtRecord.AppID)
	if err != nil {
		return nil, fmt.Errorf("auth: get roles: %w", err)
	}

	accessToken, err := s.tokenMgr.GenerateAccessToken(user, app.Slug, roles, s.cfg.JWT.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("auth: generate access token: %w", err)
	}

	rawRefreshToken := uuid.New().String()
	ttl := s.cfg.JWT.RefreshTokenTTLWeb
	if rtRecord.DeviceInfo.ClientType == string(domain.ClientTypeMobile) ||
		rtRecord.DeviceInfo.ClientType == string(domain.ClientTypeDesktop) {
		ttl = s.cfg.JWT.RefreshTokenTTLMobile
	}

	if err := s.istoreRefreshToken(ctx, user.ID, rtRecord.AppID, rawRefreshToken, rtRecord.DeviceInfo.ClientType, req.IP, req.UserAgent, ttl); err != nil {
		return nil, fmt.Errorf("auth: store new refresh token: %w", err)
	}

	userID := user.ID
	appID := rtRecord.AppID
	resType := "refresh_token"
	s.auditSvc.LogEvent(&domain.AuditLog{
		EventType:     domain.EventAuthTokenRefreshed,
		ApplicationID: &appID,
		UserID:        &userID,
		ActorID:       &userID,
		ResourceType:  &resType,
		IPAddress:     req.IP,
		UserAgent:     req.UserAgent,
		Success:       true,
	})

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWT.AccessTokenTTL.Seconds()),
	}, nil
}

// Logout revokes all active refresh tokens for the user+app.
func (s *AuthServiceI) Logout(ctx context.Context, claims *domain.Claims, appKey, ip, ua string) error {
	app, err := s.appRepo.FindBySecretKey(ctx, appKey)
	if err != nil || app == nil || !app.IsActive {
		return ErrApplicationNotFound
	}

	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return ErrInvalidCredentials
	}

	if err := s.refreshPGRepo.RevokeAllForUser(ctx, userID, app.ID); err != nil {
		return fmt.Errorf("auth: revoke all refresh tokens: %w", err)
	}

	appID := app.ID
	resType := "refresh_token"
	s.auditSvc.LogEvent(&domain.AuditLog{
		EventType:     domain.EventAuthLogout,
		ApplicationID: &appID,
		UserID:        &userID,
		ActorID:       &userID,
		ResourceType:  &resType,
		IPAddress:     ip,
		UserAgent:     ua,
		Success:       true,
	})
	return nil
}

// ChangePassword validates the current password and sets a new one.
func (s *AuthServiceI) ChangePassword(ctx context.Context, claims *domain.Claims, req ChangePasswordRequest) error {
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return ErrInvalidCredentials
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return ErrInvalidCredentials
	}

	currentNFC := norm.NFC.String(req.CurrentPassword)
	newNFC := norm.NFC.String(req.NewPassword)

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentNFC)); err != nil {
		return ErrInvalidCredentials
	}

	if err := validatePasswordPolicyInternal(newNFC); err != nil {
		return err
	}

	hashes, err := s.pwdHistoryRepo.GetLastN(ctx, userID, s.cfg.Security.PasswordHistory)
	if err != nil {
		return fmt.Errorf("auth: get password history: %w", err)
	}
	for _, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(newNFC)) == nil {
			return ErrPasswordReused
		}
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newNFC), s.cfg.Security.BcryptCost)
	if err != nil {
		return fmt.Errorf("auth: hash new password: %w", err)
	}

	if err := s.pwdHistoryRepo.Add(ctx, userID, user.PasswordHash); err != nil {
		return fmt.Errorf("auth: add to password history: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(newHash)); err != nil {
		return fmt.Errorf("auth: update password: %w", err)
	}

	s.auditSvc.LogEvent(&domain.AuditLog{
		EventType: domain.EventAuthPasswordChanged,
		UserID:    &userID,
		ActorID:   &userID,
		Success:   true,
		IPAddress: req.IP,
		UserAgent: req.UserAgent,
	})
	return nil
}

// validatePasswordPolicyInternal is a private alias so AuthServiceI can call it
// without a circular dependency on ValidatePasswordPolicy from auth_service.go.
func validatePasswordPolicyInternal(password string) error {
	if utf8.RuneCountInString(password) < 10 {
		return fmt.Errorf("%w: password must be at least 10 characters", ErrPasswordPolicy)
	}

	var hasUpper, hasDigit, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSymbol = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("%w: password must contain at least one uppercase letter", ErrPasswordPolicy)
	}
	if !hasDigit {
		return fmt.Errorf("%w: password must contain at least one number", ErrPasswordPolicy)
	}
	if !hasSymbol {
		return fmt.Errorf("%w: password must contain at least one special character", ErrPasswordPolicy)
	}
	return nil
}

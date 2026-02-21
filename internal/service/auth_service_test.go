package service_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/enunezf/sentinel/internal/config"
	"github.com/enunezf/sentinel/internal/domain"
	redisrepo "github.com/enunezf/sentinel/internal/repository/redis"
	"github.com/enunezf/sentinel/internal/service"
	"github.com/enunezf/sentinel/internal/token"
)

// ---- mock implementations ----

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *mockUserRepo) UpdateFailedAttempts(ctx context.Context, userID uuid.UUID, attempts int, lockedUntil *time.Time, lockoutCount int, lockoutDate *time.Time) error {
	return m.Called(ctx, userID, attempts, lockedUntil, lockoutCount, lockoutDate).Error(0)
}
func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, hash string) error {
	return m.Called(ctx, userID, hash).Error(0)
}

type mockAppRepo struct{ mock.Mock }

func (m *mockAppRepo) FindBySecretKey(ctx context.Context, secretKey string) (*domain.Application, error) {
	args := m.Called(ctx, secretKey)
	if a := args.Get(0); a != nil {
		return a.(*domain.Application), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockAppRepo) FindBySlug(ctx context.Context, slug string) (*domain.Application, error) {
	args := m.Called(ctx, slug)
	if a := args.Get(0); a != nil {
		return a.(*domain.Application), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockRefreshPGRepo struct{ mock.Mock }

func (m *mockRefreshPGRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockRefreshPGRepo) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if rt := args.Get(0); rt != nil {
		return rt.(*domain.RefreshToken), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockRefreshPGRepo) FindByRawToken(ctx context.Context, rawToken string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, rawToken)
	if rt := args.Get(0); rt != nil {
		return rt.(*domain.RefreshToken), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockRefreshPGRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockRefreshPGRepo) RevokeAllForUser(ctx context.Context, userID, appID uuid.UUID) error {
	return m.Called(ctx, userID, appID).Error(0)
}

type mockRefreshRedisRepo struct{ mock.Mock }

func (m *mockRefreshRedisRepo) Set(ctx context.Context, rawToken string, data redisrepo.RefreshTokenData, ttl time.Duration) error {
	return m.Called(ctx, rawToken, data, ttl).Error(0)
}
func (m *mockRefreshRedisRepo) Get(ctx context.Context, rawToken string) (*redisrepo.RefreshTokenData, error) {
	args := m.Called(ctx, rawToken)
	if d := args.Get(0); d != nil {
		return d.(*redisrepo.RefreshTokenData), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockRefreshRedisRepo) Delete(ctx context.Context, rawToken string) error {
	return m.Called(ctx, rawToken).Error(0)
}

type mockPwdHistoryRepo struct{ mock.Mock }

func (m *mockPwdHistoryRepo) GetLastN(ctx context.Context, userID uuid.UUID, n int) ([]string, error) {
	args := m.Called(ctx, userID, n)
	return args.Get(0).([]string), args.Error(1)
}
func (m *mockPwdHistoryRepo) Add(ctx context.Context, userID uuid.UUID, hash string) error {
	return m.Called(ctx, userID, hash).Error(0)
}

type mockUserRoleRepo struct{ mock.Mock }

func (m *mockUserRoleRepo) GetActiveRoleNamesForUserApp(ctx context.Context, userID, appID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, userID, appID)
	return args.Get(0).([]string), args.Error(1)
}

type mockAuditSvc struct{ mock.Mock }

func (m *mockAuditSvc) LogEvent(entry *domain.AuditLog) { m.Called(entry) }

// ---- test fixtures ----

var (
	testAppID  = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	testUserID = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
)

func makeTestConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessTokenTTL:        60 * time.Minute,
			RefreshTokenTTLWeb:    168 * time.Hour,
			RefreshTokenTTLMobile: 720 * time.Hour,
		},
		Security: config.SecurityConfig{
			MaxFailedAttempts: 5,
			LockoutDuration:   15 * time.Minute,
			BcryptCost:        4, // Low cost for tests.
			PasswordHistory:   5,
		},
	}
}

func makeTestApp() *domain.Application {
	return &domain.Application{
		ID:        testAppID,
		Name:      "Test App",
		Slug:      "test-app",
		SecretKey: "valid-app-key",
		IsActive:  true,
	}
}

func makeTestUser(t *testing.T, password string) *domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	require.NoError(t, err)
	return &domain.User{
		ID:           testUserID,
		Username:     "jperez",
		Email:        "jperez@sodexo.com",
		PasswordHash: string(hash),
		IsActive:     true,
	}
}

// newTestAuthService creates an AuthServiceIface with all mocked dependencies.
func newTestAuthService(
	userRepo service.UserRepositoryIface,
	appRepo service.ApplicationRepositoryIface,
	pgRefresh service.RefreshTokenPGRepositoryIface,
	redisRefresh service.RefreshTokenRedisRepositoryIface,
	pwdHistory service.PasswordHistoryRepositoryIface,
	userRoles service.UserRoleRepositoryIface,
	auditSvc service.AuditServiceIface,
	cfg *config.Config,
) *service.AuthServiceI {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	mgr := token.NewManagerFromKey(privKey)
	return service.NewAuthServiceI(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, mgr, auditSvc, cfg)
}

// ---- Login tests ----

func TestLogin_Success(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	userRepo.On("UpdateLastLogin", mock.Anything, user.ID).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{"chef"}, nil)
	pgRefresh.On("Create", mock.Anything, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	resp, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, 3600, resp.ExpiresIn)
	assert.Equal(t, user.Username, resp.User.Username)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	userRepo.On("UpdateFailedAttempts", mock.Anything, user.ID, 1, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "WRONG_PASSWORD",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})

	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestLogin_AccountLocked(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	lockedUntil := time.Now().Add(10 * time.Minute)
	user := makeTestUser(t, "S3cur3P@ss!")
	user.LockedUntil = &lockedUntil

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrAccountLocked)
}

func TestLogin_PermanentLock(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	// permanent lock: lockout_count >= 3 and locked_until = nil, but lockout_date set.
	lockoutDate := time.Now().UTC()
	user := makeTestUser(t, "S3cur3P@ss!")
	user.LockoutCount = 3
	user.LockoutDate = &lockoutDate
	user.LockedUntil = nil

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrAccountLocked)
}

func TestLogin_AccountInactive(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	user.IsActive = false

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrAccountInactive)
}

func TestLogin_InvalidAppKey(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	appRepo.On("FindBySecretKey", mock.Anything, "invalid-key").Return(nil, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "invalid-key",
	})
	assert.ErrorIs(t, err, service.ErrApplicationNotFound)
}

func TestLogin_InvalidClientType(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "tablet", // invalid
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrInvalidClientType)
}

func TestLogin_MissingClientType(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "", // missing
		AppKey:     "valid-app-key",
	})
	// Missing client_type is treated as invalid (not in valid enum).
	assert.ErrorIs(t, err, service.ErrInvalidClientType)
}

func TestLogin_IncrementFailedAttempts(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	user.FailedAttempts = 2 // pre-existing 2 failures
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	// After failure, attempts should be 3 (2+1), not yet locked (< 5).
	userRepo.On("UpdateFailedAttempts", mock.Anything, user.ID, 3, (*time.Time)(nil), 0, (*time.Time)(nil)).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "WRONG",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
	userRepo.AssertCalled(t, "UpdateFailedAttempts", mock.Anything, user.ID, 3, mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_LockAt5Attempts(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	user.FailedAttempts = 4 // 5th attempt is the locking one
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	// On 5th failure: locked_until must be non-nil.
	userRepo.On("UpdateFailedAttempts", mock.Anything, user.ID, 5, mock.MatchedBy(func(t *time.Time) bool {
		return t != nil && time.Until(*t) > 0
	}), mock.Anything, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "WRONG",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
	// Verify UpdateFailedAttempts was called with a non-nil locked_until.
	userRepo.AssertCalled(t, "UpdateFailedAttempts", mock.Anything, user.ID, 5, mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_PermanentLockAt3Lockouts(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	// User has 2 lockouts already today — this 5th failure triggers the 3rd lockout.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	user := makeTestUser(t, "S3cur3P@ss!")
	user.FailedAttempts = 4
	user.LockoutCount = 2
	user.LockoutDate = &today
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	// On 3rd lockout: locked_until must be nil (permanent).
	userRepo.On("UpdateFailedAttempts", mock.Anything, user.ID, 5, (*time.Time)(nil), 3, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "WRONG",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
	// locked_until must be nil for permanent lock.
	userRepo.AssertCalled(t, "UpdateFailedAttempts", mock.Anything, user.ID, 5, (*time.Time)(nil), 3, mock.Anything)
}

func TestLogin_ResetOnSuccess(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	user.FailedAttempts = 3
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	// UpdateLastLogin resets failed_attempts and locked_until.
	userRepo.On("UpdateLastLogin", mock.Anything, user.ID).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{}, nil)
	pgRefresh.On("Create", mock.Anything, mock.Anything).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	require.NoError(t, err)
	userRepo.AssertCalled(t, "UpdateLastLogin", mock.Anything, user.ID)
}

func TestLogin_ClientType_Web_TTL(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	userRepo.On("UpdateLastLogin", mock.Anything, user.ID).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{}, nil)
	// Web TTL = 168h.
	pgRefresh.On("Create", mock.Anything, mock.MatchedBy(func(rt *domain.RefreshToken) bool {
		diff := rt.ExpiresAt.Sub(time.Now())
		return diff > 167*time.Hour && diff < 169*time.Hour
	})).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(d time.Duration) bool {
		return d == 168*time.Hour
	})).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	require.NoError(t, err)
}

func TestLogin_ClientType_Mobile_TTL(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	userRepo.On("UpdateLastLogin", mock.Anything, user.ID).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{}, nil)
	// Mobile TTL = 720h.
	pgRefresh.On("Create", mock.Anything, mock.MatchedBy(func(rt *domain.RefreshToken) bool {
		diff := rt.ExpiresAt.Sub(time.Now())
		return diff > 719*time.Hour && diff < 721*time.Hour
	})).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(d time.Duration) bool {
		return d == 720*time.Hour
	})).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "mobile",
		AppKey:     "valid-app-key",
	})
	require.NoError(t, err)
}

func TestLogin_ClientType_Desktop_TTL(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "S3cur3P@ss!")
	app := makeTestApp()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "jperez").Return(user, nil)
	userRepo.On("UpdateLastLogin", mock.Anything, user.ID).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{}, nil)
	// Desktop TTL = 720h (same as mobile).
	pgRefresh.On("Create", mock.Anything, mock.MatchedBy(func(rt *domain.RefreshToken) bool {
		diff := rt.ExpiresAt.Sub(time.Now())
		return diff > 719*time.Hour && diff < 721*time.Hour
	})).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(d time.Duration) bool {
		return d == 720*time.Hour
	})).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "desktop",
		AppKey:     "valid-app-key",
	})
	require.NoError(t, err)
}

// ---- Refresh tests ----

func buildRefreshToken(userID, appID uuid.UUID, rawToken string, isRevoked bool, expiresAt time.Time) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		AppID:     appID,
		TokenHash: "some-bcrypt-hash",
		DeviceInfo: domain.DeviceInfo{
			ClientType: "web",
		},
		ExpiresAt: expiresAt,
		IsRevoked: isRevoked,
	}
}

func TestRefresh_Success(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	user := makeTestUser(t, "S3cur3P@ss!")
	rawToken := uuid.New().String()
	rt := buildRefreshToken(user.ID, app.ID, rawToken, false, time.Now().Add(48*time.Hour))

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	redisRefresh.On("Get", mock.Anything, rawToken).Return(&redisrepo.RefreshTokenData{
		UserID:     user.ID.String(),
		AppID:      app.ID.String(),
		TokenHash:  rt.TokenHash,
		ClientType: "web",
	}, nil)
	pgRefresh.On("FindByHash", mock.Anything, rt.TokenHash).Return(rt, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	pgRefresh.On("Revoke", mock.Anything, rt.ID).Return(nil)
	redisRefresh.On("Delete", mock.Anything, rawToken).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{"chef"}, nil)
	pgRefresh.On("Create", mock.Anything, mock.Anything).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	resp, err := svc.Refresh(context.Background(), service.RefreshRequest{
		RefreshToken: rawToken,
		AppKey:       "valid-app-key",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.NotEqual(t, rawToken, resp.RefreshToken, "new refresh token must differ from old one")
}

func TestRefresh_InvalidToken(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	redisRefresh.On("Get", mock.Anything, "nonexistent-token").Return(nil, nil)
	pgRefresh.On("FindByRawToken", mock.Anything, "nonexistent-token").Return(nil, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Refresh(context.Background(), service.RefreshRequest{
		RefreshToken: "nonexistent-token",
		AppKey:       "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrTokenInvalid)
}

func TestRefresh_RevokedToken(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	user := makeTestUser(t, "S3cur3P@ss!")
	rawToken := uuid.New().String()
	rt := buildRefreshToken(user.ID, app.ID, rawToken, true, time.Now().Add(48*time.Hour)) // is_revoked = true

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	redisRefresh.On("Get", mock.Anything, rawToken).Return(&redisrepo.RefreshTokenData{TokenHash: rt.TokenHash}, nil)
	pgRefresh.On("FindByHash", mock.Anything, rt.TokenHash).Return(rt, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Refresh(context.Background(), service.RefreshRequest{
		RefreshToken: rawToken,
		AppKey:       "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrTokenRevoked)
}

func TestRefresh_ExpiredToken(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	user := makeTestUser(t, "S3cur3P@ss!")
	rawToken := uuid.New().String()
	rt := buildRefreshToken(user.ID, app.ID, rawToken, false, time.Now().Add(-1*time.Hour)) // expired

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	redisRefresh.On("Get", mock.Anything, rawToken).Return(&redisrepo.RefreshTokenData{TokenHash: rt.TokenHash}, nil)
	pgRefresh.On("FindByHash", mock.Anything, rt.TokenHash).Return(rt, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Refresh(context.Background(), service.RefreshRequest{
		RefreshToken: rawToken,
		AppKey:       "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrTokenExpired)
}

func TestRefresh_DoubleUse(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	user := makeTestUser(t, "S3cur3P@ss!")
	rawToken := uuid.New().String()

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	// Second use: Redis returns nil (deleted after rotation), PG returns revoked.
	redisRefresh.On("Get", mock.Anything, rawToken).Return(nil, nil)
	revokedRT := buildRefreshToken(user.ID, app.ID, rawToken, true, time.Now().Add(48*time.Hour))
	pgRefresh.On("FindByRawToken", mock.Anything, rawToken).Return(revokedRT, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Refresh(context.Background(), service.RefreshRequest{
		RefreshToken: rawToken,
		AppKey:       "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrTokenRevoked)
}

// ---- Logout tests ----

func TestLogout_Success(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	pgRefresh.On("RevokeAllForUser", mock.Anything, testUserID, app.ID).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	claims := &domain.Claims{Sub: testUserID.String(), App: app.Slug}
	err := svc.Logout(context.Background(), claims, "valid-app-key", "127.0.0.1", "test")
	require.NoError(t, err)
	pgRefresh.AssertCalled(t, "RevokeAllForUser", mock.Anything, testUserID, app.ID)
}

// ---- ChangePassword tests ----

func TestChangePassword_Success(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "OldP@ssw0rd!")
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	pwdHistory.On("GetLastN", mock.Anything, user.ID, 5).Return([]string{}, nil)
	pwdHistory.On("Add", mock.Anything, user.ID, user.PasswordHash).Return(nil)
	userRepo.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	claims := &domain.Claims{Sub: user.ID.String()}
	err := svc.ChangePassword(context.Background(), claims, service.ChangePasswordRequest{
		CurrentPassword: "OldP@ssw0rd!",
		NewPassword:     "N3wS3cur3P@ss!",
	})
	require.NoError(t, err)
	userRepo.AssertCalled(t, "UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string"))
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "OldP@ssw0rd!")
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	claims := &domain.Claims{Sub: user.ID.String()}
	err := svc.ChangePassword(context.Background(), claims, service.ChangePasswordRequest{
		CurrentPassword: "WRONG_CURRENT",
		NewPassword:     "N3wS3cur3P@ss!",
	})
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestChangePassword_PolicyViolation(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	user := makeTestUser(t, "OldP@ssw0rd!")
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	claims := &domain.Claims{Sub: user.ID.String()}
	err := svc.ChangePassword(context.Background(), claims, service.ChangePasswordRequest{
		CurrentPassword: "OldP@ssw0rd!",
		NewPassword:     "weak", // fails policy
	})
	assert.ErrorIs(t, err, service.ErrPasswordPolicy)
}

func TestChangePassword_Reused(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	currentPwd := "OldP@ssw0rd!"
	user := makeTestUser(t, currentPwd)

	// Create a history hash for the "new" password (pretend it was used before).
	newPwd := "N3wS3cur3P@ss!"
	oldHash, _ := bcrypt.GenerateFromPassword([]byte(newPwd), 4)

	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	pwdHistory.On("GetLastN", mock.Anything, user.ID, 5).Return([]string{string(oldHash)}, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	claims := &domain.Claims{Sub: user.ID.String()}
	err := svc.ChangePassword(context.Background(), claims, service.ChangePasswordRequest{
		CurrentPassword: currentPwd,
		NewPassword:     newPwd,
	})
	assert.ErrorIs(t, err, service.ErrPasswordReused)
}

func TestChangePassword_AllowedAfter5(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	currentPwd := "OldP@ssw0rd!"
	user := makeTestUser(t, currentPwd)

	// The history contains the last 5 passwords; the target new password is older (6th+).
	// The GetLastN returns only 5, so the 6th is not checked -> should be accepted.
	newPwd := "N3wS3cur3P@ss!"
	// Make 5 hashes of totally different passwords (not matching newPwd).
	history := make([]string, 5)
	for i := range history {
		h, _ := bcrypt.GenerateFromPassword([]byte("DifferentPwd!12345_"+string(rune('0'+i))), 4)
		history[i] = string(h)
	}

	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	pwdHistory.On("GetLastN", mock.Anything, user.ID, 5).Return(history, nil)
	pwdHistory.On("Add", mock.Anything, user.ID, user.PasswordHash).Return(nil)
	userRepo.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	claims := &domain.Claims{Sub: user.ID.String()}
	err := svc.ChangePassword(context.Background(), claims, service.ChangePasswordRequest{
		CurrentPassword: currentPwd,
		NewPassword:     newPwd,
	})
	require.NoError(t, err, "password not in last 5 must be accepted")
}

// ---- user not found in redis fallback ----

func TestRefresh_NotInRedis_FallsBackToPG(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	user := makeTestUser(t, "S3cur3P@ss!")
	rawToken := uuid.New().String()
	rt := buildRefreshToken(user.ID, app.ID, rawToken, false, time.Now().Add(48*time.Hour))

	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	// Redis returns empty data (no token_hash).
	redisRefresh.On("Get", mock.Anything, rawToken).Return(&redisrepo.RefreshTokenData{}, nil)
	// Falls back to PG raw scan.
	pgRefresh.On("FindByRawToken", mock.Anything, rawToken).Return(rt, nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	pgRefresh.On("Revoke", mock.Anything, rt.ID).Return(nil)
	redisRefresh.On("Delete", mock.Anything, rawToken).Return(nil)
	userRoles.On("GetActiveRoleNamesForUserApp", mock.Anything, user.ID, app.ID).Return([]string{}, nil)
	pgRefresh.On("Create", mock.Anything, mock.Anything).Return(nil)
	redisRefresh.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	resp, err := svc.Refresh(context.Background(), service.RefreshRequest{
		RefreshToken: rawToken,
		AppKey:       "valid-app-key",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
}

// ---- login user not found ----

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	app := makeTestApp()
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(app, nil)
	userRepo.On("FindByUsername", mock.Anything, "unknown").Return(nil, nil)
	auditSvc.On("LogEvent", mock.Anything).Return()

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "unknown",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

// Verify ErrInvalidClientType wraps are distinct from ErrApplicationNotFound.
func TestLogin_InactiveApp(t *testing.T) {
	userRepo := &mockUserRepo{}
	appRepo := &mockAppRepo{}
	pgRefresh := &mockRefreshPGRepo{}
	redisRefresh := &mockRefreshRedisRepo{}
	pwdHistory := &mockPwdHistoryRepo{}
	userRoles := &mockUserRoleRepo{}
	auditSvc := &mockAuditSvc{}
	cfg := makeTestConfig()

	inactiveApp := makeTestApp()
	inactiveApp.IsActive = false
	appRepo.On("FindBySecretKey", mock.Anything, "valid-app-key").Return(inactiveApp, nil)

	svc := newTestAuthService(userRepo, appRepo, pgRefresh, redisRefresh, pwdHistory, userRoles, auditSvc, cfg)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username:   "jperez",
		Password:   "S3cur3P@ss!",
		ClientType: "web",
		AppKey:     "valid-app-key",
	})
	assert.ErrorIs(t, err, service.ErrApplicationNotFound)
}

// errSentinel is used for propagation checks.
var errSentinel = errors.New("sentinel-error")

package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the full application configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JWTConfig       `yaml:"jwt"`
	Security  SecurityConfig  `yaml:"security"`
	Bootstrap BootstrapConfig `yaml:"bootstrap"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port                    int           `yaml:"port"`
	ReadTimeout             time.Duration `yaml:"read_timeout"`
	WriteTimeout            time.Duration `yaml:"write_timeout"`
	GracefulShutdownTimeout time.Duration `yaml:"graceful_shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Name            string        `yaml:"name"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// DSN returns a pgx-compatible connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		d.Host, d.Port, d.Name, d.User, d.Password,
	)
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// JWTConfig holds JWT signing key paths and TTL settings.
type JWTConfig struct {
	PrivateKeyPath        string        `yaml:"private_key_path"`
	PublicKeyPath         string        `yaml:"public_key_path"`
	AccessTokenTTL        time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTLWeb    time.Duration `yaml:"refresh_token_ttl_web"`
	RefreshTokenTTLMobile time.Duration `yaml:"refresh_token_ttl_mobile"`
}

// SecurityConfig holds security policy settings.
type SecurityConfig struct {
	MaxFailedAttempts int           `yaml:"max_failed_attempts"`
	LockoutDuration   time.Duration `yaml:"lockout_duration"`
	BcryptCost        int           `yaml:"bcrypt_cost"`
	PasswordHistory   int           `yaml:"password_history"`
}

// BootstrapConfig holds initial admin credentials.
type BootstrapConfig struct {
	AdminUser     string `yaml:"admin_user"`
	AdminPassword string `yaml:"admin_password"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
	Output string `yaml:"output"` // stdout, stderr (default: stdout)
}

// envVarRegexp matches ${VAR_NAME} placeholders in YAML values.
var envVarRegexp = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnvVars replaces ${VAR} placeholders with environment variable values.
func expandEnvVars(s string) string {
	return envVarRegexp.ReplaceAllStringFunc(s, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		val := os.Getenv(name)
		return val
	})
}

// rawConfig is used to unmarshal the YAML before env expansion.
type rawConfig map[string]interface{}

// Load reads the config from path, expands environment variable placeholders,
// and validates that all required fields are present.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot read file %q: %w", path, err)
	}

	// Expand ${VAR} placeholders using environment variables.
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: cannot parse YAML: %w", err)
	}

	// Apply defaults.
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Server.GracefulShutdownTimeout == 0 {
		cfg.Server.GracefulShutdownTimeout = 15 * time.Second
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 50
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 10
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.JWT.AccessTokenTTL == 0 {
		cfg.JWT.AccessTokenTTL = 60 * time.Minute
	}
	if cfg.JWT.RefreshTokenTTLWeb == 0 {
		cfg.JWT.RefreshTokenTTLWeb = 168 * time.Hour
	}
	if cfg.JWT.RefreshTokenTTLMobile == 0 {
		cfg.JWT.RefreshTokenTTLMobile = 720 * time.Hour
	}
	if cfg.Security.BcryptCost == 0 {
		cfg.Security.BcryptCost = 12
	}
	if cfg.Security.MaxFailedAttempts == 0 {
		cfg.Security.MaxFailedAttempts = 5
	}
	if cfg.Security.LockoutDuration == 0 {
		cfg.Security.LockoutDuration = 15 * time.Minute
	}
	if cfg.Security.PasswordHistory == 0 {
		cfg.Security.PasswordHistory = 5
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate checks that all required fields have non-empty values.
func validate(cfg *Config) error {
	var errs []string

	if cfg.Database.Host == "" {
		errs = append(errs, "database.host (DB_HOST) is required")
	}
	if cfg.Database.Name == "" {
		errs = append(errs, "database.name (DB_NAME) is required")
	}
	if cfg.Database.User == "" {
		errs = append(errs, "database.user (DB_USER) is required")
	}
	if cfg.Database.Password == "" {
		errs = append(errs, "database.password (DB_PASSWORD) is required")
	}
	if cfg.Redis.Addr == "" {
		errs = append(errs, "redis.addr (REDIS_ADDR) is required")
	}
	if cfg.JWT.PrivateKeyPath == "" {
		errs = append(errs, "jwt.private_key_path (JWT_PRIVATE_KEY_PATH) is required")
	}
	if cfg.JWT.PublicKeyPath == "" {
		errs = append(errs, "jwt.public_key_path (JWT_PUBLIC_KEY_PATH) is required")
	}
	if cfg.Bootstrap.AdminUser == "" {
		errs = append(errs, "bootstrap.admin_user (BOOTSTRAP_ADMIN_USER) is required")
	}
	if cfg.Bootstrap.AdminPassword == "" {
		errs = append(errs, "bootstrap.admin_password (BOOTSTRAP_ADMIN_PASSWORD) is required")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed:\n  - " + strings.Join(errs, "\n  - "))
	}

	return nil
}

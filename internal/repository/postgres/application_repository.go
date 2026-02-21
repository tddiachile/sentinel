package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/enunezf/sentinel/internal/domain"
)

// ApplicationRepository handles persistence of Application entities.
type ApplicationRepository struct {
	db *pgxpool.Pool
}

// NewApplicationRepository creates a new ApplicationRepository.
func NewApplicationRepository(db *pgxpool.Pool) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

const appSelectFields = `id, name, slug, secret_key, is_active, created_at, updated_at`

func scanApp(row pgx.Row) (*domain.Application, error) {
	var a domain.Application
	err := row.Scan(&a.ID, &a.Name, &a.Slug, &a.SecretKey, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// FindBySecretKey returns the application matching the given secret_key, or nil.
func (r *ApplicationRepository) FindBySecretKey(ctx context.Context, secretKey string) (*domain.Application, error) {
	q := `SELECT ` + appSelectFields + ` FROM applications WHERE secret_key = $1`
	row := r.db.QueryRow(ctx, q, secretKey)
	a, err := scanApp(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("app_repo: find by secret key: %w", err)
	}
	return a, nil
}

// FindBySlug returns the application matching the slug, or nil.
func (r *ApplicationRepository) FindBySlug(ctx context.Context, slug string) (*domain.Application, error) {
	q := `SELECT ` + appSelectFields + ` FROM applications WHERE slug = $1`
	row := r.db.QueryRow(ctx, q, slug)
	a, err := scanApp(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("app_repo: find by slug: %w", err)
	}
	return a, nil
}

// Create inserts a new application.
func (r *ApplicationRepository) Create(ctx context.Context, app *domain.Application) error {
	const q = `
		INSERT INTO applications (id, name, slug, secret_key, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`
	_, err := r.db.Exec(ctx, q, app.ID, app.Name, app.Slug, app.SecretKey, app.IsActive)
	if err != nil {
		return fmt.Errorf("app_repo: create: %w", err)
	}
	return nil
}

// ExistsAny returns true if at least one application exists.
func (r *ApplicationRepository) ExistsAny(ctx context.Context) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM applications`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("app_repo: exists any: %w", err)
	}
	return count > 0, nil
}

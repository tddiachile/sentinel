package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/enunezf/sentinel/internal/domain"
)

// UserCostCenterRepository manages user_cost_centers assignments.
type UserCostCenterRepository struct {
	db *pgxpool.Pool
}

// NewUserCostCenterRepository creates a new UserCostCenterRepository.
func NewUserCostCenterRepository(db *pgxpool.Pool) *UserCostCenterRepository {
	return &UserCostCenterRepository{db: db}
}

// Assign creates a new user_cost_center assignment.
func (r *UserCostCenterRepository) Assign(ctx context.Context, ucc *domain.UserCostCenter) error {
	const q = `
		INSERT INTO user_cost_centers (user_id, cost_center_id, application_id, granted_by, valid_from, valid_until)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, cost_center_id) DO UPDATE
		SET valid_from = EXCLUDED.valid_from, valid_until = EXCLUDED.valid_until, granted_by = EXCLUDED.granted_by`
	_, err := r.db.Exec(ctx, q,
		ucc.UserID, ucc.CostCenterID, ucc.ApplicationID,
		ucc.GrantedBy, ucc.ValidFrom, ucc.ValidUntil,
	)
	if err != nil {
		return fmt.Errorf("user_cc_repo: assign: %w", err)
	}
	return nil
}

// ListForUser returns all active user_cost_centers for a user.
func (r *UserCostCenterRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserCostCenter, error) {
	const q = `
		SELECT ucc.user_id, ucc.cost_center_id, ucc.application_id, ucc.granted_by,
		       ucc.valid_from, ucc.valid_until, cc.code, cc.name
		FROM user_cost_centers ucc
		JOIN cost_centers cc ON cc.id = ucc.cost_center_id
		WHERE ucc.user_id = $1
		ORDER BY cc.code ASC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("user_cc_repo: list for user: %w", err)
	}
	defer rows.Close()

	var result []*domain.UserCostCenter
	for rows.Next() {
		var ucc domain.UserCostCenter
		if err := rows.Scan(&ucc.UserID, &ucc.CostCenterID, &ucc.ApplicationID, &ucc.GrantedBy,
			&ucc.ValidFrom, &ucc.ValidUntil, &ucc.CostCenterCode, &ucc.CostCenterName); err != nil {
			return nil, fmt.Errorf("user_cc_repo: scan: %w", err)
		}
		result = append(result, &ucc)
	}
	return result, rows.Err()
}

// GetActiveCodesForUserApp returns cost center codes active for a user+app.
func (r *UserCostCenterRepository) GetActiveCodesForUserApp(ctx context.Context, userID, appID uuid.UUID) ([]string, error) {
	const q = `
		SELECT cc.code
		FROM user_cost_centers ucc
		JOIN cost_centers cc ON cc.id = ucc.cost_center_id
		WHERE ucc.user_id = $1
		  AND ucc.application_id = $2
		  AND ucc.valid_from <= NOW()
		  AND (ucc.valid_until IS NULL OR ucc.valid_until > NOW())`
	rows, err := r.db.Query(ctx, q, userID, appID)
	if err != nil {
		return nil, fmt.Errorf("user_cc_repo: get active codes: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("user_cc_repo: scan code: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

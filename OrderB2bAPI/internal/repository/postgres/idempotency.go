package postgres

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
)

type idempotencyKeyRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewIdempotencyKeyRepository creates a new idempotency key repository
func NewIdempotencyKeyRepository(db *sql.DB, logger *zap.Logger) *idempotencyKeyRepository {
	return &idempotencyKeyRepository{
		db:     db,
		logger: logger,
	}
}

func (r *idempotencyKeyRepository) GetByKey(ctx context.Context, key string) (*domain.IdempotencyKey, error) {
	query := `
		SELECT key, partner_id, supplier_order_id, request_hash, created_at
		FROM idempotency_keys
		WHERE key = $1
	`

	var idempotencyKey domain.IdempotencyKey

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&idempotencyKey.Key,
		&idempotencyKey.PartnerID,
		&idempotencyKey.SupplierOrderID,
		&idempotencyKey.RequestHash,
		&idempotencyKey.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.Error("Failed to get idempotency key", zap.Error(err))
		return nil, err
	}

	return &idempotencyKey, nil
}

func (r *idempotencyKeyRepository) Create(ctx context.Context, key *domain.IdempotencyKey) error {
	query := `
		INSERT INTO idempotency_keys (key, partner_id, supplier_order_id, request_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()
	if key.CreatedAt.IsZero() {
		key.CreatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		key.Key,
		key.PartnerID,
		key.SupplierOrderID,
		key.RequestHash,
		key.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create idempotency key", zap.Error(err))
		return err
	}

	return nil
}

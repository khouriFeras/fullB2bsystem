package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type partnerRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewPartnerRepository creates a new partner repository
func NewPartnerRepository(db *sql.DB, logger *zap.Logger) *partnerRepository {
	return &partnerRepository{
		db:     db,
		logger: logger,
	}
}

func apiKeyLookupHash(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(h[:])
}

func (r *partnerRepository) GetByAPIKeyHash(ctx context.Context, apiKey string) (*domain.Partner, error) {
	// Prefer direct lookup by api_key_lookup (SHA256 hex) when set; then verify with bcrypt.
	lookupKey := apiKeyLookupHash(apiKey)
	queryByLookup := `
		SELECT id, name, api_key_hash, webhook_url, collection_handle, is_active, created_at, updated_at
		FROM partners
		WHERE is_active = true AND api_key_lookup = $1
	`
	var partner domain.Partner
	var webhookURL, collectionHandle sql.NullString
	err := r.db.QueryRowContext(ctx, queryByLookup, lookupKey).Scan(
		&partner.ID,
		&partner.Name,
		&partner.APIKeyHash,
		&webhookURL,
		&collectionHandle,
		&partner.IsActive,
		&partner.CreatedAt,
		&partner.UpdatedAt,
	)
	if err == nil {
		// Found by lookup; verify with bcrypt and return
		if bcrypt.CompareHashAndPassword([]byte(partner.APIKeyHash), []byte(apiKey)) == nil {
			if webhookURL.Valid {
				partner.WebhookURL = &webhookURL.String
			}
			if collectionHandle.Valid && collectionHandle.String != "" {
				partner.CollectionHandle = &collectionHandle.String
			}
			return &partner, nil
		}
		r.logger.Debug("API key lookup found partner but bcrypt verification failed", zap.String("partner_id", partner.ID.String()))
		// Fall through to iterate (will likely still fail)
	} else if err != sql.ErrNoRows {
		r.logger.Debug("API key lookup query error (falling back to iterate)", zap.Error(err))
	}
	// No row or column not yet present: fall back to iterating all active partners (legacy)
	query := `
		SELECT id, name, api_key_hash, webhook_url, collection_handle, is_active, created_at, updated_at
		FROM partners
		WHERE is_active = true
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to query partners", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
		var p domain.Partner
		var wh, ch sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.APIKeyHash, &wh, &ch, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(p.APIKeyHash), []byte(apiKey)) == nil {
			if wh.Valid {
				p.WebhookURL = &wh.String
			}
			if ch.Valid && ch.String != "" {
				p.CollectionHandle = &ch.String
			}
			return &p, nil
		}
	}

	r.logger.Info("API key did not match any partner",
		zap.Int("active_partners_checked", count),
		zap.Int("api_key_len", len(apiKey)),
		zap.String("lookup_key_prefix", safePrefix(lookupKey, 8)))
	return nil, &errors.ErrUnauthorized{Message: "invalid API key"}
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (r *partnerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Partner, error) {
	query := `
		SELECT id, name, api_key_hash, webhook_url, collection_handle, is_active, created_at, updated_at
		FROM partners
		WHERE id = $1
	`

	var partner domain.Partner
	var webhookURL, collectionHandle sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&partner.ID,
		&partner.Name,
		&partner.APIKeyHash,
		&webhookURL,
		&collectionHandle,
		&partner.IsActive,
		&partner.CreatedAt,
		&partner.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errors.ErrNotFound{Resource: "partner", ID: id.String()}
	}
	if err != nil {
		r.logger.Error("Failed to get partner by ID", zap.Error(err))
		return nil, err
	}

	if webhookURL.Valid {
		partner.WebhookURL = &webhookURL.String
	}
	if collectionHandle.Valid && collectionHandle.String != "" {
		partner.CollectionHandle = &collectionHandle.String
	}

	return &partner, nil
}

func (r *partnerRepository) List(ctx context.Context) ([]*domain.Partner, error) {
	query := `
		SELECT id, name, collection_handle, is_active, created_at
		FROM partners
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list partners", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var partners []*domain.Partner
	for rows.Next() {
		var p domain.Partner
		var collHandle sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &collHandle, &p.IsActive, &p.CreatedAt); err != nil {
			r.logger.Error("Failed to scan partner", zap.Error(err))
			return nil, err
		}
		if collHandle.Valid && collHandle.String != "" {
			p.CollectionHandle = &collHandle.String
		}
		partners = append(partners, &p)
	}
	return partners, rows.Err()
}

func (r *partnerRepository) ListWithCollectionHandle(ctx context.Context) ([]*domain.Partner, error) {
	query := `
		SELECT id, name, collection_handle, is_active, created_at
		FROM partners
		WHERE collection_handle IS NOT NULL AND TRIM(collection_handle) != ''
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list partners with collection", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var partners []*domain.Partner
	for rows.Next() {
		var p domain.Partner
		var collHandle sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &collHandle, &p.IsActive, &p.CreatedAt); err != nil {
			r.logger.Error("Failed to scan partner", zap.Error(err))
			return nil, err
		}
		if collHandle.Valid && collHandle.String != "" {
			p.CollectionHandle = &collHandle.String
		}
		partners = append(partners, &p)
	}
	return partners, rows.Err()
}

func (r *partnerRepository) Create(ctx context.Context, partner *domain.Partner) error {
	query := `
		INSERT INTO partners (id, name, api_key_hash, api_key_lookup, webhook_url, collection_handle, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	if partner.ID == uuid.Nil {
		partner.ID = uuid.New()
	}
	if partner.CreatedAt.IsZero() {
		partner.CreatedAt = now
	}
	if partner.UpdatedAt.IsZero() {
		partner.UpdatedAt = now
	}

	var apiKeyLookup interface{}
	if partner.APIKeyLookup != "" {
		apiKeyLookup = partner.APIKeyLookup
	}
	_, err := r.db.ExecContext(ctx, query,
		partner.ID,
		partner.Name,
		partner.APIKeyHash,
		apiKeyLookup,
		partner.WebhookURL,
		partner.CollectionHandle,
		partner.IsActive,
		partner.CreatedAt,
		partner.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create partner", zap.Error(err))
		return err
	}

	return nil
}

func (r *partnerRepository) Update(ctx context.Context, partner *domain.Partner) error {
	query := `
		UPDATE partners
		SET name = $2, api_key_hash = $3, webhook_url = $4, collection_handle = $5, is_active = $6, updated_at = $7
		WHERE id = $1
	`

	partner.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		partner.ID,
		partner.Name,
		partner.APIKeyHash,
		partner.WebhookURL,
		partner.CollectionHandle,
		partner.IsActive,
		partner.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to update partner", zap.Error(err))
		return err
	}

	return nil
}

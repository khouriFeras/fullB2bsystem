package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
)

type orderEventRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewOrderEventRepository creates a new order event repository
func NewOrderEventRepository(db *sql.DB, logger *zap.Logger) *orderEventRepository {
	return &orderEventRepository{
		db:     db,
		logger: logger,
	}
}

func (r *orderEventRepository) Create(ctx context.Context, event *domain.OrderEvent) error {
	query := `
		INSERT INTO order_events (id, supplier_order_id, event_type, event_data, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}

	var eventDataJSON []byte
	var err error
	if event.EventData != nil {
		eventDataJSON, err = json.Marshal(event.EventData)
		if err != nil {
			return err
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		event.SupplierOrderID,
		event.EventType,
		eventDataJSON,
		event.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to create order event", zap.Error(err))
		return err
	}

	return nil
}

func (r *orderEventRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderEvent, error) {
	query := `
		SELECT id, supplier_order_id, event_type, event_data, created_at
		FROM order_events
		WHERE supplier_order_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		r.logger.Error("Failed to get order events by order ID", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var events []*domain.OrderEvent
	for rows.Next() {
		var event domain.OrderEvent
		var eventDataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.SupplierOrderID,
			&event.EventType,
			&eventDataJSON,
			&event.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if len(eventDataJSON) > 0 {
			if err := json.Unmarshal(eventDataJSON, &event.EventData); err != nil {
				return nil, err
			}
		}

		events = append(events, &event)
	}

	return events, rows.Err()
}

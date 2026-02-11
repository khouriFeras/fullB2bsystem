package postgres

import (
	"database/sql"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/repository"
)

// NewRepositories creates a new set of repositories
func NewRepositories(db *sql.DB, logger *zap.Logger) *repository.Repositories {
	return &repository.Repositories{
		Partner:           NewPartnerRepository(db, logger),
		SupplierOrder:     NewSupplierOrderRepository(db, logger),
		SupplierOrderItem: NewSupplierOrderItemRepository(db, logger),
		IdempotencyKey:    NewIdempotencyKeyRepository(db, logger),
		SKUMapping:        NewSKUMappingRepository(db, logger),
		PartnerSKUMapping: NewPartnerSKUMappingRepository(db, logger),
		OrderEvent:        NewOrderEventRepository(db, logger),
	}
}

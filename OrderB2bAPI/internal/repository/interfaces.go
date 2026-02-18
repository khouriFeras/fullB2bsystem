package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jafarshop/b2bapi/internal/domain"
)

// PartnerRepository defines partner data access methods
type PartnerRepository interface {
	GetByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Partner, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Partner, error)
	List(ctx context.Context) ([]*domain.Partner, error)
	ListWithCollectionHandle(ctx context.Context) ([]*domain.Partner, error)
	Create(ctx context.Context, partner *domain.Partner) error
	Update(ctx context.Context, partner *domain.Partner) error
}

// SupplierOrderRepository defines supplier order data access methods
type SupplierOrderRepository interface {
	Create(ctx context.Context, order *domain.SupplierOrder) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.SupplierOrder, error)
	GetByPartnerIDAndPartnerOrderID(ctx context.Context, partnerID uuid.UUID, partnerOrderID string) (*domain.SupplierOrder, error)
	GetByPartnerOrderID(ctx context.Context, partnerOrderID string) (*domain.SupplierOrder, error)
	GetByShopifyOrderID(ctx context.Context, shopifyOrderID string) (*domain.SupplierOrder, error)
	Update(ctx context.Context, order *domain.SupplierOrder) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus, rejectionReason *string) error
	UpdateStatusFromShopify(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
	UpdateTracking(ctx context.Context, id uuid.UUID, carrier, trackingNumber, trackingURL *string) error
	UpdateLastDeliveryStatus(ctx context.Context, id uuid.UUID, status int, statusLabel, waybill, imageURL string) error
	UpdateShopifyDraftOrderID(ctx context.Context, id uuid.UUID, draftOrderID int64) error
	UpdateShopifyOrderID(ctx context.Context, id uuid.UUID, orderID string) error
	ListByPartnerID(ctx context.Context, partnerID uuid.UUID, limit, offset int) ([]*domain.SupplierOrder, error)
	ListByPartnerIDAndStatus(ctx context.Context, partnerID uuid.UUID, status domain.OrderStatus, limit, offset int) ([]*domain.SupplierOrder, error)
	ListByStatus(ctx context.Context, status domain.OrderStatus, limit, offset int) ([]*domain.SupplierOrder, error)
}

// SupplierOrderItemRepository defines order item data access methods
type SupplierOrderItemRepository interface {
	Create(ctx context.Context, item *domain.SupplierOrderItem) error
	CreateBatch(ctx context.Context, items []*domain.SupplierOrderItem) error
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.SupplierOrderItem, error)
}

// IdempotencyKeyRepository defines idempotency key data access methods
type IdempotencyKeyRepository interface {
	GetByKey(ctx context.Context, key string) (*domain.IdempotencyKey, error)
	Create(ctx context.Context, key *domain.IdempotencyKey) error
}

// SKUMappingRepository defines SKU mapping data access methods
type SKUMappingRepository interface {
	GetBySKU(ctx context.Context, sku string) (*domain.SKUMapping, error)
	GetActiveSKUs(ctx context.Context) ([]string, error)
	Create(ctx context.Context, mapping *domain.SKUMapping) error
	Update(ctx context.Context, mapping *domain.SKUMapping) error
	Upsert(ctx context.Context, mapping *domain.SKUMapping) error
	GetAllActive(ctx context.Context) ([]*domain.SKUMapping, error)
}

// OrderEventRepository defines order event data access methods
type OrderEventRepository interface {
	Create(ctx context.Context, event *domain.OrderEvent) error
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.OrderEvent, error)
}

// PartnerSKUMappingRepository defines partner-scoped SKU mapping data access
type PartnerSKUMappingRepository interface {
	GetBySKUAndPartner(ctx context.Context, partnerID uuid.UUID, sku string) (*domain.PartnerSKUMapping, error)
	ListByPartnerID(ctx context.Context, partnerID uuid.UUID) ([]*domain.PartnerSKUMapping, error)
	Upsert(ctx context.Context, m *domain.PartnerSKUMapping) error
	UpsertBatch(ctx context.Context, partnerID uuid.UUID, mappings []*domain.PartnerSKUMapping) error
}

// Repositories aggregates all repositories
type Repositories struct {
	Partner           PartnerRepository
	SupplierOrder     SupplierOrderRepository
	SupplierOrderItem SupplierOrderItemRepository
	IdempotencyKey    IdempotencyKeyRepository
	SKUMapping        SKUMappingRepository
	PartnerSKUMapping PartnerSKUMappingRepository
	OrderEvent        OrderEventRepository
}

package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/repository"
	"github.com/jafarshop/b2bapi/pkg/errors"
)

type skuService struct {
	repos  *repository.Repositories
	logger *zap.Logger
}

// NewSKUService creates a new SKU service
func NewSKUService(repos *repository.Repositories, logger *zap.Logger) *skuService {
	return &skuService{
		repos:  repos,
		logger: logger,
	}
}

// CheckCartForSupplierSKUs checks if cart contains at least one supplier SKU (global sku_mappings; legacy)
// Returns: hasSupplierSKU, supplierItems map (SKU -> mapping), error
func (s *skuService) CheckCartForSupplierSKUs(
	ctx context.Context,
	items []CartItem,
) (bool, map[string]*domain.SKUMapping, error) {
	supplierItems := make(map[string]*domain.SKUMapping)

	for _, item := range items {
		mapping, err := s.repos.SKUMapping.GetBySKU(ctx, item.SKU)
		if err != nil {
			if _, isNotFound := err.(*errors.ErrNotFound); isNotFound {
				continue
			}
			s.logger.Warn("Error checking SKU mapping", zap.String("sku", item.SKU), zap.Error(err))
			continue
		}
		if mapping.IsActive {
			supplierItems[item.SKU] = mapping
		}
	}
	return len(supplierItems) > 0, supplierItems, nil
}

// CheckCartForPartnerSKUs checks if cart contains at least one SKU in this partner's catalog (partner_sku_mappings)
// Returns: hasPartnerSKU, partnerItems map (SKU -> mapping), error. Use for cart submit (per-partner catalog).
func (s *skuService) CheckCartForPartnerSKUs(
	ctx context.Context,
	partnerID uuid.UUID,
	items []CartItem,
) (bool, map[string]*domain.PartnerSKUMapping, error) {
	partnerItems := make(map[string]*domain.PartnerSKUMapping)
	for _, item := range items {
		mapping, err := s.repos.PartnerSKUMapping.GetBySKUAndPartner(ctx, partnerID, item.SKU)
		if err != nil {
			if _, isNotFound := err.(*errors.ErrNotFound); isNotFound {
				continue
			}
			s.logger.Warn("Error checking partner SKU mapping", zap.String("sku", item.SKU), zap.String("partner_id", partnerID.String()), zap.Error(err))
			continue
		}
		if mapping.IsActive {
			partnerItems[item.SKU] = mapping
		}
	}
	return len(partnerItems) > 0, partnerItems, nil
}

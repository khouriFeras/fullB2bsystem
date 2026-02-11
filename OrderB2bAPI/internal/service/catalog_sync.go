package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/jafarshop/b2bapi/internal/config"
	"github.com/jafarshop/b2bapi/internal/domain"
	"github.com/jafarshop/b2bapi/internal/productb2b"
	"github.com/jafarshop/b2bapi/internal/repository"
)

const syncInterval = 10 * time.Minute

var catalogSyncMu sync.Mutex

// RunCatalogSyncOnce runs catalog sync once for all partners with collection_handle.
// For each partner, fetches products from ProductB2B and upserts into partner_sku_mappings.
// Does not block; logs errors per partner.
func RunCatalogSyncOnce(ctx context.Context, cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) {
	if cfg.ProductB2B.BaseURL == "" || cfg.ProductB2B.ServiceKey == "" {
		logger.Debug("Catalog sync skipped: PRODUCT_B2B_URL or PRODUCT_B2B_SERVICE_API_KEY not set")
		return
	}
	partners, err := repos.Partner.ListWithCollectionHandle(ctx)
	if err != nil {
		logger.Error("Catalog sync: failed to list partners with collection", zap.Error(err))
		return
	}
	if len(partners) == 0 {
		logger.Debug("Catalog sync: no partners with collection_handle")
		return
	}
	client := productb2b.NewClient(cfg.ProductB2B.BaseURL, cfg.ProductB2B.ServiceKey, logger)
	for _, p := range partners {
		if p.CollectionHandle == nil || *p.CollectionHandle == "" {
			continue
		}
		handle := *p.CollectionHandle
		cursor := ""
		limit := 50
		var allMappings []*domain.PartnerSKUMapping
		for {
			body, err := client.GetCatalogProducts(ctx, handle, cursor, limit)
			if err != nil {
				logger.Warn("Catalog sync: ProductB2B request failed for partner", zap.String("partner_id", p.ID.String()), zap.String("collection_handle", handle), zap.Error(err))
				break
			}
			resp, err := productb2b.ParseCatalogProducts(body)
			if err != nil {
				logger.Warn("Catalog sync: parse failed for partner", zap.String("partner_id", p.ID.String()), zap.Error(err))
				break
			}
			for _, node := range resp.Data {
				_, rows := productb2b.ExtractProductVariantInfos(node)
				for _, r := range rows {
					m := &domain.PartnerSKUMapping{
						PartnerID:        p.ID,
						SKU:              r.SKU,
						ShopifyProductID: r.ShopifyProductID,
						ShopifyVariantID: r.ShopifyVariantID,
						IsActive:         true,
					}
					if r.Title != "" {
						m.Title = &r.Title
					}
					if r.Price != "" {
						m.Price = &r.Price
					}
					m.ImageURL = r.ImageURL
					allMappings = append(allMappings, m)
				}
			}
			hasNext := false
			if resp.Pagination != nil {
				if v, ok := resp.Pagination["hasNextPage"].(bool); ok && v {
					hasNext = true
				}
				if v, ok := resp.Pagination["nextCursor"].(string); ok && v != "" {
					cursor = v
				}
				if cursor == "" {
					if v, ok := resp.Pagination["endCursor"].(string); ok && v != "" {
						cursor = v
					}
				}
			}
			if !hasNext || cursor == "" {
				break
			}
		}
		if len(allMappings) > 0 {
			if err := repos.PartnerSKUMapping.UpsertBatch(ctx, p.ID, allMappings); err != nil {
				logger.Warn("Catalog sync: upsert failed for partner", zap.String("partner_id", p.ID.String()), zap.Error(err))
			} else {
				logger.Info("Catalog sync: synced partner catalog", zap.String("partner_id", p.ID.String()), zap.Int("mappings", len(allMappings)))
			}
		}
	}
}

// RunCatalogSyncLoop runs sync once, then every syncInterval. Call from a goroutine.
func RunCatalogSyncLoop(ctx context.Context, cfg *config.Config, repos *repository.Repositories, logger *zap.Logger) {
	catalogSyncMu.Lock()
	RunCatalogSyncOnce(ctx, cfg, repos, logger)
	catalogSyncMu.Unlock()

	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			catalogSyncMu.Lock()
			RunCatalogSyncOnce(ctx, cfg, repos, logger)
			catalogSyncMu.Unlock()
		}
	}
}

package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const webhookTimeout = 10 * time.Second

// NotifyDeliveryUpdate sends a delivery update payload to the partner's webhook URL.
// It is intended to be called in a goroutine so the API response is not blocked.
// Payload should include: partner_id, order_id (optional), partner_order_id (optional),
// shipping_address, shipment, and optionally event ("delivery_status" or "order_shipped").
func NotifyDeliveryUpdate(webhookURL string, payload map[string]interface{}, logger *zap.Logger) {
	if webhookURL == "" {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("Webhook: failed to marshal delivery payload", zap.Error(err))
		return
	}
	client := &http.Client{Timeout: webhookTimeout}
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		logger.Warn("Webhook: failed to create request", zap.String("url", webhookURL), zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Webhook: delivery notification request failed", zap.String("url", webhookURL), zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("Webhook: delivery notification returned non-2xx",
			zap.String("url", webhookURL), zap.Int("status", resp.StatusCode))
		return
	}
	logger.Info("Webhook: delivery notification sent", zap.String("url", webhookURL), zap.Int("status", resp.StatusCode))
}

# JafarShop B2B API - Integration Checklist

Use this checklist to ensure a smooth integration with the JafarShop B2B API.

## Pre-Integration

### Requirements Gathering

- [ ] **API Key Received**
  - [ ] API key provided by JafarShop
  - [ ] API key stored securely (not in code)
  - [ ] Base URL confirmed (production/staging)

- [ ] **Environment Setup**
  - [ ] Development environment ready
  - [ ] Testing environment configured
  - [ ] Production environment prepared

- [ ] **Documentation Review**
  - [ ] Read [PARTNER_GUIDE.md](./PARTNER_GUIDE.md)
  - [ ] Reviewed [API_DOCUMENTATION.md](./API_DOCUMENTATION.md)
  - [ ] Understood SKU mapping concept
  - [ ] Understood order lifecycle

## Integration Phase

### Authentication Setup

- [ ] **API Key Integration**
  - [ ] API key stored in environment variable
  - [ ] Authorization header implemented: `Bearer {api_key}`
  - [ ] API key never logged or exposed
  - [ ] Error handling for 401 responses

### Cart Submission

- [ ] **Request Implementation**
  - [ ] POST `/v1/carts/submit` endpoint implemented
  - [ ] All required fields included
  - [ ] JSON payload properly formatted
  - [ ] Content-Type header set to `application/json`

- [ ] **Required Fields**
  - [ ] `partner_order_id` (unique per order)
  - [ ] `items` array (at least 1 item)
  - [ ] `customer.name`
  - [ ] `shipping.street`, `city`, `postal_code`, `country`
  - [ ] `totals.subtotal`, `tax`, `shipping`, `total`

- [ ] **Optional Fields**
  - [ ] `customer.phone` (if available)
  - [ ] `shipping.state` (if applicable)
  - [ ] `payment_status` (if known)
  - [ ] `payment_method` (if known)
  - [ ] `items[].product_url` (if available)

- [ ] **Idempotency**
  - [ ] Unique UUID generated for each order
  - [ ] `Idempotency-Key` header included
  - [ ] Idempotency key stored with order (for retries)

### Response Handling

- [ ] **Success Responses**
  - [ ] 200 OK: Order created, store `supplier_order_id`
  - [ ] 204 No Content: No JafarShop products (expected, not error)

- [ ] **Error Handling**
  - [ ] 401: Invalid API key (check credentials)
  - [ ] 422: Validation error (check field requirements)
  - [ ] 409: Idempotency conflict (generate new key)
  - [ ] 500: Server error (retry with backoff)

### Order Status Tracking

- [ ] **Status Polling**
  - [ ] GET `/v1/orders/{id}` endpoint implemented
  - [ ] Polling logic implemented
  - [ ] Status update logic in your system
  - [ ] Polling frequency configured (5-10 min for pending)

- [ ] **Status Handling**
  - [ ] `PENDING_CONFIRMATION` → Continue polling
  - [ ] `CONFIRMED` → Order approved, update your system
  - [ ] `SHIPPED` → Order shipped, update with tracking
  - [ ] `REJECTED` → Handle rejection (check `rejection_reason`)
  - [ ] `CANCELLED` → Handle cancellation

## Testing Phase

### Unit Testing

- [ ] **Request Building**
  - [ ] Test request payload construction
  - [ ] Test field validation
  - [ ] Test required vs optional fields

- [ ] **Response Parsing**
  - [ ] Test success response parsing
  - [ ] Test error response parsing
  - [ ] Test 204 No Content handling

### Integration Testing

- [ ] **Cart Submission Tests**
  - [ ] Test with mapped SKU (should create order)
  - [ ] Test with unmapped SKU only (should return 204)
  - [ ] Test with mixed cart (mapped + unmapped)
  - [ ] Test with all required fields
  - [ ] Test with optional fields

- [ ] **Idempotency Tests**
  - [ ] Test same key + same payload (should return existing order)
  - [ ] Test same key + different payload (should return 409)
  - [ ] Test different keys (should create new orders)

- [ ] **Error Handling Tests**
  - [ ] Test with invalid API key (401)
  - [ ] Test with missing required fields (422)
  - [ ] Test with invalid data types (422)
  - [ ] Test with wrong order ID (404)

- [ ] **Status Polling Tests**
  - [ ] Test status retrieval
  - [ ] Test status transitions
  - [ ] Test polling timeout handling

### End-to-End Testing

- [ ] **Complete Order Flow**
  - [ ] Submit order → Get order ID
  - [ ] Poll status → See PENDING_CONFIRMATION
  - [ ] Wait for confirmation → See CONFIRMED
  - [ ] Wait for shipping → See SHIPPED with tracking

- [ ] **Edge Cases**
  - [ ] Large order (many items)
  - [ ] Order with special characters
  - [ ] Order with international address
  - [ ] Order retry after network failure

## Pre-Launch Checklist

### Security

- [ ] API key stored securely (not hardcoded)
- [ ] API key not in version control
- [ ] HTTPS used in production
- [ ] Error messages don't expose sensitive data

### Error Handling

- [ ] All error cases handled
- [ ] Retry logic implemented (for transient errors)
- [ ] Error logging implemented
- [ ] Error notifications configured

### Monitoring

- [ ] Request/response logging (without API keys)
- [ ] Error rate monitoring
- [ ] Order status tracking
- [ ] Alerting for critical errors

### Documentation

- [ ] Integration documented internally
- [ ] Error handling procedures documented
- [ ] Support contacts documented
- [ ] Rollback plan documented

## Go-Live

### Final Checks

- [ ] Production API key confirmed
- [ ] Production base URL confirmed
- [ ] All tests passing
- [ ] Monitoring in place
- [ ] Support team notified

### Launch Steps

1. [ ] Switch to production API key
2. [ ] Switch to production base URL
3. [ ] Submit test order
4. [ ] Verify order appears in JafarShop system
5. [ ] Monitor for first 24 hours
6. [ ] Verify status updates working

### Post-Launch

- [ ] Monitor error rates
- [ ] Monitor order processing times
- [ ] Verify status updates timely
- [ ] Check for any integration issues
- [ ] Gather feedback from operations team

## Maintenance

### Ongoing

- [ ] Monitor API health
- [ ] Review error logs regularly
- [ ] Update integration as API evolves
- [ ] Keep API key secure
- [ ] Test after API updates

### When to Contact Support

- [ ] High error rates (> 5%)
- [ ] Orders not appearing in JafarShop
- [ ] Status updates not working
- [ ] API key issues
- [ ] Unexpected behavior

**Support Email:** Feras.jafarShop@gmail.com

## Quick Reference

### Required Headers

```
Authorization: Bearer {api_key}
Content-Type: application/json
Idempotency-Key: {uuid}  (recommended)
```

### Minimum Request

```json
{
  "partner_order_id": "unique-order-id",
  "items": [{"sku": "SKU", "title": "Title", "price": 10, "quantity": 1}],
  "customer": {"name": "Name"},
  "shipping": {"street": "Street", "city": "City", "postal_code": "12345", "country": "JO"},
  "totals": {"subtotal": 10, "tax": 0, "shipping": 0, "total": 10}
}
```

### Status Codes

- `200` - Success
- `204` - No JafarShop products (expected)
- `401` - Invalid API key
- `422` - Validation error
- `409` - Idempotency conflict

---

**Last Updated:** 2026-01-21

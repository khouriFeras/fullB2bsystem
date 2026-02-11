# Testing Guide - B2B API

This guide will help you test the B2B API as Zain Shop before the actual partnership.

## Prerequisites

1. ✅ Server is running on `http://localhost:8080`
2. ✅ PostgreSQL database is running
3. ✅ Shopify credentials are configured in `.env`

## Step 1: Create Test Partner (Zain Shop)

Create a partner account for testing:

```bash
go run cmd/create-partner/main.go "Zain Shop" "zain-test-api-key-2024"
```

**Save the API key!** You'll need it for testing.

Example output:
```
 Partner created successfully!

Partner ID: 550e8400-e29b-41d4-a716-446655440000
Partner Name: Zain Shop
API Key: zain-test-api-key-2024

  IMPORTANT: Save this API key securely! You won't be able to see it again.

Use this API key in the Authorization header:
Authorization: Bearer zain-test-api-key-2024
```

## Step 2: Add Test SKU Mappings

Add some of your Shopify product SKUs to the system. You need:
- SKU (from your Shopify product)
- Shopify Product ID
- Shopify Variant ID

To find these in Shopify:
1. Go to your Shopify admin
2. Open a product
3. The Product ID is in the URL: `/admin/products/123456789` → ID is `123456789`
4. Click on a variant → Variant ID is in the URL or use GraphQL to query

Add a SKU mapping:

```bash
go run cmd/add-sku/main.go "JS-PROD-001" 123456789 987654321
```

Repeat for each product SKU you want to test with.

## Step 3: Test Cart Submission

### Using PowerShell (Invoke-WebRequest)

```powershell
$headers = @{
    "Authorization" = "Bearer zain-test-api-key-2024"
    "Idempotency-Key" = "test-order-001"
    "Content-Type" = "application/json"
}

$body = @{
    partner_order_id = "ZAIN-ORDER-001"
    items = @(
        @{
            sku = "JS-PROD-001"
            title = "JafarShop Product"
            price = 29.99
            quantity = 2
            product_url = "https://zainshop.com/product/js-prod-001"
        },
        @{
            sku = "OTHER-001"
            title = "Other Product (Not JafarShop)"
            price = 19.99
            quantity = 1
            product_url = "https://zainshop.com/product/other-001"
        }
    )
    customer = @{
        name = "feras"
        phone = "+1234567890"
    }
    shipping = @{
        street = "amman street"
        city = "amman"
        state = "khalda"
        postal_code = "10001"
        country = "jordan"
    }
    totals = @{
        subtotal = 79.97
        tax = 6.40
        shipping = 5.00
        total = 91.37
    }
    payment_status = "paid"
} | ConvertTo-Json -Depth 10

Invoke-WebRequest -Uri "http://localhost:8080/v1/carts/submit" `
    -Method POST `
    -Headers $headers `
    -Body $body `
    -UseBasicParsing
```

### Using curl (if available)

```bash
curl -X POST http://localhost:8080/v1/carts/submit \
  -H "Authorization: Bearer zain-test-api-key-2024" \
  -H "Idempotency-Key: test-order-001" \
  -H "Content-Type: application/json" \
  -d '{
    "partner_order_id": "ZAIN-ORDER-001",
    "items": [
      {
        "sku": "JS-PROD-001",
        "title": "JafarShop Product",
        "price": 29.99,
        "quantity": 2,
        "product_url": "https://zainshop.com/product/js-prod-001"
      }
    ],
    "customer": {
      "name": "John Doe",
      "phone": "+1234567890"
    },
    "shipping": {
      "street": "123 Main Street",
      "city": "New York",
      "state": "NY",
      "postal_code": "10001",
      "country": "US"
    },
    "totals": {
      "subtotal": 59.98,
      "tax": 4.80,
      "shipping": 5.00,
      "total": 69.78
    },
    "payment_status": "paid"
  }'
```

### Expected Responses

**Success (200 OK):**
```json
{
  "supplier_order_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING_CONFIRMATION"
}
```

**No Supplier SKUs (204 No Content):**
- If cart doesn't contain any of your SKUs, returns 204
- No response body

## Step 4: Check Order Status

```powershell
$headers = @{
    "Authorization" = "Bearer zain-test-api-key-2024"
}

Invoke-WebRequest -Uri "http://localhost:8080/v1/orders/550e8400-e29b-41d4-a716-446655440000" `
    -Method GET `
    -Headers $headers `
    -UseBasicParsing
```

## Step 5: Admin Actions (Confirm/Reject/Ship)

### Confirm Order

```powershell
$headers = @{
    "Authorization" = "Bearer zain-test-api-key-2024"
    "Content-Type" = "application/json"
}

Invoke-WebRequest -Uri "http://localhost:8080/v1/admin/orders/550e8400-e29b-41d4-a716-446655440000/confirm" `
    -Method POST `
    -Headers $headers `
    -UseBasicParsing
```

### Ship Order

```powershell
$body = @{
    carrier = "Standard Shipping"
    tracking_number = "TRACK123456789"
    tracking_url = "https://example.com/track/TRACK123456789"
} | ConvertTo-Json

Invoke-WebRequest -Uri "http://localhost:8080/v1/admin/orders/550e8400-e29b-41d4-a716-446655440000/ship" `
    -Method POST `
    -Headers $headers `
    -Body $body `
    -UseBasicParsing
```

## Testing Scenarios

### Scenario 1: Cart with Supplier SKU
- ✅ Should create order
- ✅ Should create Shopify draft order
- ✅ Should return 200 with order ID

### Scenario 2: Cart without Supplier SKU
- ✅ Should return 204 No Content
- ✅ Should not create order

### Scenario 3: Duplicate Request (Idempotency)
- ✅ Send same request twice with same `Idempotency-Key`
- ✅ Second request should return same order (not create duplicate)

### Scenario 4: Different Payload, Same Key
- ✅ Send request with same `Idempotency-Key` but different items
- ✅ Should return 409 Conflict

## Quick Test Checklist

- [ ] Partner created
- [ ] At least one SKU mapping added
- [ ] Cart submission works (200 response)
- [ ] Order status endpoint works
- [ ] Admin confirm works
- [ ] Admin ship works
- [ ] Idempotency works (duplicate request returns same order)

## When Real Partnership Starts

1. Create production partner account:
   ```bash
   go run cmd/create-partner/main.go "Zain Shop" "<secure-production-api-key>"
   ```

2. Sync all SKU mappings from Shopify (you may want to create a script for this)

3. Provide Zain Shop with:
   - API endpoint URL (your production server)
   - API key
   - API documentation (`API_DOCUMENTATION.md`)

4. They integrate on their end - your API is ready!

## Troubleshooting

**401 Unauthorized:**
- Check API key is correct
- Verify partner is active in database

**204 No Content:**
- Cart doesn't contain any supplier SKUs
- Check SKU mappings are correct

**500 Internal Server Error:**
- Check server logs
- Verify Shopify credentials are correct
- Check database connection

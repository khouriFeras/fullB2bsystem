# OrderB2bAPI – Endpoint Tests

Copy and paste these commands into your terminal to test each endpoint. Use `https://api.jafarshop.com` for production, or `http://localhost:8081` for local.

Replace `YOUR_PARTNER_API_KEY` with a valid partner API key (from `create-partner`).

**Windows PowerShell:** Use `curl.exe` instead of `curl`. Or use `Invoke-RestMethod` examples below.

---

## Quick run: test all endpoints

```powershell
$env:PARTNER_API_KEY = "YOUR_PARTNER_API_KEY"
.\test-all-endpoints.ps1
```

---

## 1. Health check

Liveness check. No auth.

```powershell
Invoke-RestMethod -Uri "http://localhost:8081/health"
# or
curl.exe -s "http://localhost:8081/health"
```

Expected: `{"status":"ok"}` (HTTP 200)

---

## 2. Catalog products (partner auth)

Partner's product catalog (paginated). Requires Bearer token.

```powershell
$headers = @{ "Authorization" = "Bearer YOUR_PARTNER_API_KEY" }
Invoke-RestMethod -Uri "http://localhost:8081/v1/catalog/products?limit=5" -Headers $headers
# or
curl.exe -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" "http://localhost:8081/v1/catalog/products?limit=5"
```

Expected: JSON with `data` array, `pagination`, `meta` (HTTP 200)

---

## 3. Cart submit (create order)

Submits a cart and creates an order. Requires `Idempotency-Key`. SKU must exist in partner catalog.

```powershell
$headers = @{
    "Authorization" = "Bearer YOUR_PARTNER_API_KEY"
    "Content-Type" = "application/json"
    "Idempotency-Key" = "unique-key-$(Get-Date -Format 'yyyyMMddHHmmss')"
}
$body = @{
    partner_order_id = "test-order-001"
    items = @(
        @{ sku = "REAL-SKU"; title = "Product"; price = 10; quantity = 1 }
    )
    customer = @{
        first_name = "Test"
        last_name = "User"
        email = "test@example.com"
        phone_number = "+962700000000"
    }
    shipping = @{ city = "Amman"; area = ""; address = "123 St"; postal_code = ""; country = "Jordan" }
    totals = @{ subtotal = 10; tax = 0; shipping = 0; total = 10 }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Uri "http://localhost:8081/v1/carts/submit" -Method POST -Headers $headers -Body $body -ContentType "application/json"
```

Expected: `{"supplier_order_id":"...","status":"PENDING_CONFIRMATION",...}` (HTTP 200)  
Or 422 if SKU not in partner catalog.

---

## 4. List orders (admin)

Lists partner's orders with pagination. Optional `status` filter.

```powershell
$headers = @{ "Authorization" = "Bearer YOUR_PARTNER_API_KEY" }
Invoke-RestMethod -Uri "http://localhost:8081/v1/admin/orders?limit=10" -Headers $headers
# With status filter
Invoke-RestMethod -Uri "http://localhost:8081/v1/admin/orders?status=PENDING_CONFIRMATION&limit=5" -Headers $headers
```

Expected: JSON with `orders` array, `limit`, `offset` (HTTP 200)

---

## 5. Get order by ID

Returns a single order by supplier UUID or partner_order_id.

```powershell
$headers = @{ "Authorization" = "Bearer YOUR_PARTNER_API_KEY" }
# By supplier order UUID
Invoke-RestMethod -Uri "http://localhost:8081/v1/orders/550e8400-e29b-41d4-a716-446655440000" -Headers $headers
# By partner_order_id
Invoke-RestMethod -Uri "http://localhost:8081/v1/orders/test-order-001" -Headers $headers
```

Expected: Full order JSON (HTTP 200) or 404 if not found.

---

## 6. Delivery status

Delivery/shipment status for an order.

```powershell
$headers = @{ "Authorization" = "Bearer YOUR_PARTNER_API_KEY" }
Invoke-RestMethod -Uri "http://localhost:8081/v1/orders/ORDER_ID_OR_PARTNER_ORDER_ID/delivery-status" -Headers $headers
```

Expected: JSON with delivery info (HTTP 200) or 503 if GetDeliveryStatus service unavailable.

---

## 7. Unauthorized (no auth)

Protected endpoint without auth should return 401.

```powershell
try { Invoke-RestMethod -Uri "http://localhost:8081/v1/catalog/products?limit=1" } catch { $_.Exception.Response.StatusCode.Value__ }
```

Expected: 401

---

## One-liner: run all tests (status codes)

**PowerShell:**

```powershell
$KEY = "YOUR_PARTNER_API_KEY"
$BASE = "http://localhost:8081"
Write-Host "1. Health: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" $BASE/health
Write-Host "`n2. Catalog: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/v1/catalog/products?limit=5"
Write-Host "`n3. List orders: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/v1/admin/orders?limit=5"
Write-Host "`n4. No auth (401): " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/v1/catalog/products?limit=1"
```

---

## Prerequisites

- OrderB2bAPI running (default: `http://localhost:8081`)
- ProductB2B running (OrderB2bAPI fetches catalog from it)
- A partner created: `go run cmd/create-partner/main.go "Test Partner" "your-api-key"`

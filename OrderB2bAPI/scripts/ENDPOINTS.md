# OrderB2bAPI & ProductB2B – Endpoint test reference

Base URLs (adjust if needed):
- **OrderB2bAPI:** `http://localhost:8081`
- **ProductB2B:** `http://localhost:3000`

Use your **partner API key** for OrderB2bAPI and **PRODUCT_B2B_SERVICE_API_KEY** for ProductB2B service calls.

---

## OrderB2bAPI (port 8081)

All `/v1/*` routes require: `Authorization: Bearer <partner-api-key>`

| # | Method | Path | Auth | Description |
|---|--------|------|------|-------------|
| 1 | GET | `/health` | No | Liveness |
| 2 | GET | `/v1/catalog/products?limit=5&cursor=` | Partner | Partner's catalog (from ProductB2B or fallback) |
| 3 | POST | `/v1/carts/submit` | Partner + `Idempotency-Key` | Submit cart → create order |
| 4 | GET | `/v1/admin/orders?limit=50&offset=0&status=` | Partner | List partner's orders |
| 5 | GET | `/v1/orders/:id` | Partner | Get order by UUID or `partner_order_id` |

### One-liners (PowerShell)

```powershell
$API = "http://localhost:8081"
$KEY = "YOUR_PARTNER_API_KEY"

# 1. Health
curl.exe -s "$API/health"

# 2. Catalog
curl.exe -s -H "Authorization: Bearer $KEY" "$API/v1/catalog/products?limit=5"

# 3. Cart submit (minimal body – use a real SKU from catalog)
$body = '{"partner_order_id":"order-001","items":[{"sku":"REAL-SKU","title":"Product","price":10,"quantity":1}],"customer":{"first_name":"Test","last_name":"User","email":"t@e.com","phone_number":"+962700000000"},"shipping":{"city":"Amman","area":"","address":"123 St","postal_code":"","country":"Jordan"},"totals":{"subtotal":10,"tax":0,"shipping":0,"total":10}}'
curl.exe -s -X POST -H "Authorization: Bearer $KEY" -H "Idempotency-Key: unique-key-here" -H "Content-Type: application/json" -d $body "$API/v1/carts/submit"

# 4. List orders
curl.exe -s -H "Authorization: Bearer $KEY" "$API/v1/admin/orders?limit=5"

# 5. Get order (use id from list or partner_order_id)
curl.exe -s -H "Authorization: Bearer $KEY" "$API/v1/orders/ORDER_UUID_OR_PARTNER_ORDER_ID"
```

---

## ProductB2B (port 3000)

| # | Method | Path | Auth | Description |
|---|--------|------|------|-------------|
| 1 | GET | `/health` | No | Liveness |
| 2 | GET | `/v1/catalog/products?collection_handle=partner-zain&limit=5` | Service key | Catalog by collection (used by OrderB2bAPI) |
| 3 | GET | `/v1/catalog/products?sku=MK4820b` | Service or partner key | Single product by SKU |
| 4 | GET | `/debug/sku-lookup?sku=MK4820b` | No | SKU lookup (no auth) |
| 5 | GET | `/debug/partner-products` | No | Default collection products (uses PARTNER_COLLECTION_HANDLE) |
| 6 | GET | `/menus` | - | All menus (nested) |
| 7 | GET | `/menu-path-by-sku?sku=MK4820b` | - | Menu path for SKU |

### One-liners (ProductB2B)

```powershell
$B2B = "http://localhost:3000"
$SVC = "YOUR_PRODUCT_B2B_SERVICE_API_KEY"

# Health
curl.exe -s "$B2B/health"

# Catalog by collection (service key)
curl.exe -s -H "Authorization: Bearer $SVC" "$B2B/v1/catalog/products?collection_handle=partner-zain&limit=5"

# Debug SKU (no auth)
curl.exe -s "$B2B/debug/sku-lookup?sku=MK4820b"
```

---

## Run full test script

From `OrderB2bAPI` directory:

```powershell
.\scripts\test-endpoints.ps1
```

Edit the variables at the top of `test-endpoints.ps1` if your keys or base URLs differ.

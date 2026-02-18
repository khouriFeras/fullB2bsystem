# Full Testing Guide – Orders & Shipment Delivery

End-to-end testing for the JafarShop B2B API: create an order, confirm it in Shopify, then test delivery status via webhook and GET.

---

## Prerequisites

### 1. API key and base URL

- **Partner API key** – from JafarShop (or create via `create-partner`).
- **Base URL** – `https://api.jafarshop.com` (production) or `http://localhost:8081` (local).

### 2. SKU mapping

Cart items must use **partner SKUs** that are mapped to JafarShop products. If no item maps, `POST /v1/carts/submit` returns **204** (no JafarShop products). Use a mapped SKU (e.g. `1102`, `1824` per your partner catalog).

### 3. Server / deployment (for delivery webhook)

- **Migrations** – Ensure `last_delivery_*` columns exist (migration `000009`). From repo root: `./deploy/run-migrations.sh`.
- **OrderB2bAPI** – Set `DELIVERY_WEBHOOK_SECRET` in env (used to authenticate the internal delivery webhook).
- **GetDeliveryStatus** (optional, for Wassel) – Set `DELIVERY_WEBHOOK_SECRET` (same value as OrderB2bAPI) and `ORDER_B2B_API_URL` so it can forward webhooks. Set `WASSEL_SHARED_SECRET` for Wassel’s Bearer token.

---

## Part 1: Create an order (cart submit)

Use a unique `partner_order_id` (e.g. `ORDER-100`). The **Shopify order note** will show exactly this value.

### cURL

```bash
curl -X POST "https://api.jafarshop.com/v1/carts/submit" \
  -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "partner_order_id": "ORDER-100",
    "items": [
      {
        "sku": "1102",
        "title": "Product Name",
        "price": 10.00,
        "quantity": 1
      }
    ],
    "customer": {
      "first_name": "John",
      "last_name": "Doe",
      "email": "john@example.com",
      "phone_number": "+962700000000"
    },
    "shipping": {
      "city": "Amman",
      "area": "",
      "address": "123 Main St",
      "postal_code": "11118",
      "country": "JO"
    },
    "totals": {
      "subtotal": 10.00,
      "tax": 0.00,
      "shipping": 5.00,
      "total": 15.00
    }
  }'
```

### PowerShell

```powershell
$headers = @{
    "Authorization"   = "Bearer YOUR_PARTNER_API_KEY"
    "Content-Type"    = "application/json"
    "Idempotency-Key" = "unique-" + (Get-Date -Format 'yyyyMMddHHmmss')
}
$body = @{
    partner_order_id = "ORDER-100"
    items            = @(
        @{ sku = "1102"; title = "Product"; price = 10; quantity = 1 }
    )
    customer = @{
        first_name   = "John"
        last_name    = "Doe"
        email        = "john@example.com"
        phone_number = "+962700000000"
    }
    shipping = @{
        city        = "Amman"
        area        = ""
        address     = "123 Main St"
        postal_code = "11118"
        country     = "JO"
    }
    totals = @{ subtotal = 10; tax = 0; shipping = 5; total = 15 }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Uri "https://api.jafarshop.com/v1/carts/submit" -Method POST -Headers $headers -Body $body -ContentType "application/json"
```

### Expected response (200)

```json
{
  "supplier_order_id": "550e8400-e29b-41d4-a716-446655440000",
  "partner_order_id": "ORDER-100",
  "status": "PENDING_CONFIRMATION",
  "shopify_draft_order_id": 123456789,
  "shopify_order_id": "1040"
}
```

- **204** – No cart item SKU is in partner’s catalog mapping; use a mapped SKU.
- **422** – Validation error (missing/invalid fields).

Save `supplier_order_id` and/or `partner_order_id` and (if present) `shopify_order_id` for the next steps.

---

## Part 2: Verify in Shopify

1. In Shopify Admin, open the order created from the cart (by order number, e.g. **#1040**).
2. Check the **order note**: it should show **ORDER-100** (the `partner_order_id` you sent).

---

## Part 3: Delivery status – two ways to get data

Delivery status can come from:

1. **Stored** – A delivery webhook (internal or via Wassel → GetDeliveryStatus) already updated the order; GET returns stored status.
2. **Live** – No stored status; API calls GetDeliveryStatus (Wassel) and returns that result (if configured).

---

## Part 4: Send a delivery webhook (store status)

You can test storage either by calling the **internal webhook** directly or by having **Wassel** send to GetDeliveryStatus (which forwards to the internal webhook).

### 4a. Internal webhook (direct test)

Call OrderB2bAPI’s internal delivery webhook with the same Bearer secret as `DELIVERY_WEBHOOK_SECRET`.

**Identify the order** by:

- **Shopify order number** (e.g. `1039`, `1040`), or  
- **Partner order ID** (e.g. `ORDER-100`, `TEST-ORDER-20260216-1040`).

**cURL**

```bash
curl -s -X POST "https://api.jafarshop.com/internal/webhooks/delivery" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_DELIVERY_WEBHOOK_SECRET" \
  -d '{
    "ItemReferenceNo": "ORDER-100",
    "Status": 170,
    "Waybill": "WB123",
    "DeliveryImageUrl": "https://example.com/proof.jpg"
  }'
```

**PowerShell**

```powershell
$headers = @{
    "Content-Type"    = "application/json"
    "Authorization"   = "Bearer YOUR_DELIVERY_WEBHOOK_SECRET"
}
$body = '{"ItemReferenceNo":"ORDER-100","Status":170,"Waybill":"WB123","DeliveryImageUrl":"https://example.com/proof.jpg"}'
Invoke-RestMethod -Uri "https://api.jafarshop.com/internal/webhooks/delivery" -Method POST -Headers $headers -Body $body -ContentType "application/json"
```

**Expected response (200)**

- Order found and status stored, partner has no webhook URL:  
  `"ok": true`, `"status": "no_webhook"`, `"shipment": { "status", "status_label", ... }`
- Order found and partner has webhook:  
  `"ok": true`, `"status": "forwarded"`
- Order not found:  
  `"ok": true`, `"status": "not_found"`, `"message": "no order for ItemReferenceNo"`

After a successful store, GET delivery-status will return `"source": "stored"` (see below).

### 4b. Wassel → GetDeliveryStatus → OrderB2bAPI

1. **Wassel** sends POST to:  
   `https://webhooks.jafarshop.com/webhooks/wassel/status`  
   With header: `Authorization: Bearer <token JafarShop gave Wassel>` (see `GetDeliveryStatus/WASSEL-SETUP-FOR-PARTNER.md`).
2. **GetDeliveryStatus** validates the Bearer token and forwards the payload to OrderB2bAPI:  
   `POST {ORDER_B2B_API_URL}/internal/webhooks/delivery`  
   with `Authorization: Bearer {DELIVERY_WEBHOOK_SECRET}`.
3. OrderB2bAPI looks up the order by **Shopify order id** or **partner_order_id** (`ItemReferenceNo`), stores the last delivery status, and optionally notifies the partner’s webhook URL.

**Minimal Wassel payload**

```json
{"ItemReferenceNo": "ORDER-100", "Status": 170}
```

**With waybill and proof-of-delivery**

```json
{
  "ItemReferenceNo": "ORDER-100",
  "Status": 170,
  "Waybill": "26020021000001",
  "DeliveryImageUrl": "https://example.com/proof.jpg"
}
```

**Wassel status codes (we store and display)**  
See `GetDeliveryStatus/WASSEL-WEBHOOK-SPEC.json`. Examples: `51`–`130` (in transit), **`170` = Delivered to customer**, `180`–`210` (returns).

---

## Part 5: GET delivery status (partner auth)

Partners retrieve delivery status by **partner_order_id** or **supplier order UUID**.

### cURL

```bash
# By partner_order_id (e.g. ORDER-100)
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "https://api.jafarshop.com/v1/orders/ORDER-100/delivery-status"

# By supplier order UUID
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "https://api.jafarshop.com/v1/orders/550e8400-e29b-41d4-a716-446655440000/delivery-status"
```

### PowerShell

```powershell
$headers = @{ "Authorization" = "Bearer YOUR_PARTNER_API_KEY" }
Invoke-RestMethod -Uri "https://api.jafarshop.com/v1/orders/ORDER-100/delivery-status" -Headers $headers
```

### When status was stored from webhook (source: stored)

**200** – Body includes `"source": "stored"` and `shipment` from the last webhook:

```json
{
  "event": "delivery_status",
  "source": "stored",
  "shipment": {
    "status": 170,
    "status_label": "Delivered to customer",
    "waybill": "WB123",
    "delivery_image_url": "https://example.com/proof.jpg",
    "updated_at": "2026-02-15T12:00:00Z"
  }
}
```

### When no webhook received yet (live call to GetDeliveryStatus)

**200** – Body includes `"source": "live"` (or no `source`) and `shipment` from GetDeliveryStatus/Wassel. May also include `shipping_address`, `partner_id`.

### Other responses

- **404** – Order not found or not accessible for this partner.
- **503** – GetDeliveryStatus unavailable (only when we try live and no stored status).

---

## Part 6: Full flow checklist

| Step | Action | What to check |
|------|--------|----------------|
| 1 | Run migrations (if not done) | `./deploy/run-migrations.sh` |
| 2 | `POST /v1/carts/submit` with `partner_order_id: "ORDER-100"` and mapped SKU | 200, get `supplier_order_id` / `shopify_order_id` |
| 3 | Open order in Shopify | Note = **ORDER-100** |
| 4 | `POST /internal/webhooks/delivery` with `ItemReferenceNo: "ORDER-100"`, `Status: 170` (Bearer = `DELIVERY_WEBHOOK_SECRET`) | 200, `ok: true`, status stored |
| 5 | `GET /v1/orders/ORDER-100/delivery-status` (Bearer = partner API key) | 200, `source: "stored"`, `shipment.status: 170`, `status_label: "Delivered to customer"` |

---

## Quick reference

| Item | Value |
|------|--------|
| Cart submit | `POST /v1/carts/submit` — Auth: Bearer partner API key, Header: `Idempotency-Key` |
| Order by id | `GET /v1/orders/{id}` — `{id}` = partner_order_id or supplier order UUID |
| Delivery status | `GET /v1/orders/{id}/delivery-status` — same `{id}` |
| Internal delivery webhook | `POST /internal/webhooks/delivery` — Auth: Bearer `DELIVERY_WEBHOOK_SECRET` |
| Wassel → JafarShop | `POST https://webhooks.jafarshop.com/webhooks/wassel/status` — Auth: Bearer token from JafarShop |

---

## Related docs

- **Partner quick start:** [PARTNER_QUICK_START.md](./PARTNER_QUICK_START.md)
- **Delivery test steps (server):** [DELIVERY-STATUS-TEST-STEPS.md](./DELIVERY-STATUS-TEST-STEPS.md)
- **Wassel setup (URL + token):** [GetDeliveryStatus/WASSEL-SETUP-FOR-PARTNER.md](../GetDeliveryStatus/WASSEL-SETUP-FOR-PARTNER.md)
- **Wassel payload spec:** [GetDeliveryStatus/WASSEL-WEBHOOK-SPEC.json](../GetDeliveryStatus/WASSEL-WEBHOOK-SPEC.json)
- **Endpoint tests:** [TEST-ENDPOINTS.md](./TEST-ENDPOINTS.md)

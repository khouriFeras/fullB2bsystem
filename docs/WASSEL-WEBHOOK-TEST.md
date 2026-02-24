# Wassel webhook – test flow (you ↔ Wassel only)

Test the path **Wassel → GetDeliveryStatus → OrderB2bAPI** and confirm we store and return delivery status. Partner webhook (notify partners) can be tested later.

**All URLs below are production** (`api.jafarshop.com`, `webhooks.jafarshop.com`). Do not use localhost.

---

## Prerequisites

1. **An order in the DB** with a known reference you can use as `ItemReferenceNo`:
   - **Shopify order number** (e.g. `1040`, `1039`), or  
   - **Partner order ID** (e.g. `ORDER-100`).

   Check in DB or via API:
   ```sql
   SELECT id, partner_order_id, shopify_order_id, last_delivery_status
   FROM supplier_orders ORDER BY created_at DESC LIMIT 5;
   ```

2. **Server env**
   - **OrderB2bAPI** `.env`: `DELIVERY_WEBHOOK_SECRET` set (used when GetDeliveryStatus forwards to us).
   - **GetDeliveryStatus** env (e.g. in `GetDeliveryStatus/.env` or Docker):  
     `WASSEL_SHARED_SECRET`, `ORDER_B2B_API_URL` (e.g. `http://orderb2bapi:8080`), `DELIVERY_WEBHOOK_SECRET` (same value as OrderB2bAPI).
   - **OrderB2bAPI** (optional): `WASSEL_DEFAULT_PARTNER_ID` — UUID of a partner. When set, any webhook for an unknown `ItemReferenceNo` creates a minimal order under this partner and stores delivery status. Partners see only orders belonging to their `partner_id`, so only this default partner sees Wassel-created orders until the real order exists.

3. **Migrations** applied so `supplier_orders` has `last_delivery_*` columns (migration `000009`).

---

## Test 1: Internal webhook only (OrderB2bAPI)

Proves that **OrderB2bAPI** accepts the payload and stores it (no GetDeliveryStatus involved).

**Request**

```bash
curl -s -X POST "https://api.jafarshop.com/internal/webhooks/delivery" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_DELIVERY_WEBHOOK_SECRET" \
  -d '{"ItemReferenceNo":"ORDER-100","Status":170,"Waybill":"WB123","DeliveryImageUrl":"https://example.com/proof.jpg"}'
```

Replace:
- `YOUR_DELIVERY_WEBHOOK_SECRET` with the real secret from server `.env`.
- `ItemReferenceNo` with an order that exists: e.g. `1040` (Shopify #) or `ORDER-100` (partner_order_id).

**Expected (200)**

- Order found: `"ok": true`, `"status": "no_webhook"` (or `"forwarded"` if partner has webhook), and `"shipment": { "status": 170, "status_label": "Delivered to customer", ... }`.
- Order not found: `"ok": true`, `"status": "not_found"`.

If you get **401**: wrong or missing `DELIVERY_WEBHOOK_SECRET`.  
If you get **503**: `DELIVERY_WEBHOOK_SECRET` not set in OrderB2bAPI env.

---

## Test 2: Full path (Wassel → GetDeliveryStatus → OrderB2bAPI)

Simulate **Wassel** sending to **GetDeliveryStatus**. GetDeliveryStatus will validate Bearer and forward the same payload to OrderB2bAPI.

**Request (as if you are Wassel)**

```bash
curl -s -X POST "https://webhooks.jafarshop.com/webhooks/wassel/status" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_WASSEL_SHARED_SECRET" \
  -d '{"ItemReferenceNo":"ORDER-100","Status":170,"Waybill":"WB456"}'
```

Replace:
- `YOUR_WASSEL_SHARED_SECRET` with the value in GetDeliveryStatus env (`WASSEL_SHARED_SECRET`). This is the token you give to Wassel; for testing you use it yourself.
- `ItemReferenceNo` with the same order ref as in Test 1.

**Expected (200)**

- `{"ok":true}` from GetDeliveryStatus.  
- In the background, GetDeliveryStatus forwards to OrderB2bAPI; the order’s `last_delivery_*` is updated (same as Test 1).

If you get **401**: wrong or missing `WASSEL_SHARED_SECRET` in GetDeliveryStatus.  
If you get **503**: `WASSEL_SHARED_SECRET` not set in GetDeliveryStatus.  
If you get **400**: body must be valid JSON with `ItemReferenceNo` (string) and `Status` (integer).

After this, the order should have stored delivery status (same as after Test 1).

---

## Test 3: GET delivery-status (stored)

Confirm that **GET delivery-status** returns the stored data (no live call to Wassel).

**Request**

```bash
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "https://api.jafarshop.com/v1/orders/ORDER-100/delivery-status"
```

Use the same order ref (e.g. `ORDER-100` or the order’s UUID) and your partner API key.

**Expected (200)**

- `"source": "stored"`.
- `"shipment"` with `status`, `status_label`, `waybill`, `delivery_image_url`, `updated_at` from the last webhook (Test 1 or Test 2).

If you get **404**: wrong order id or partner doesn’t own that order.  
If you see no `source: "stored"` and instead a live response: no webhook was stored for that order; re-run Test 1 or Test 2 with the correct `ItemReferenceNo`.

---

## Test 4: Verify in DB (optional)

```sql
SELECT partner_order_id, shopify_order_id,
       last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_at
FROM supplier_orders
WHERE partner_order_id = 'ORDER-100';
```

You should see `last_delivery_status = 170`, `last_delivery_status_label = 'Delivered to customer'`, and the waybill/date you sent.

---

## Quick check: logs = success

After sending a delivery webhook (internal or via GetDeliveryStatus), **if the logs look good, you’re good.**

**OrderB2bAPI** (when it receives the internal webhook):

- No error and HTTP 200 from the webhook call → order was found and `last_delivery_*` was updated.
- If you log at info level, look for lines about delivery webhook / order update.

**GetDeliveryStatus** (when Wassel hits `POST /webhooks/wassel/status`):

- Returns 200 and `{"ok":true}` to the client.
- If it forwards to OrderB2bAPI, check OrderB2bAPI logs (above); a warning in GetDeliveryStatus means the forward failed (e.g. wrong URL or secret).

So: **trigger the webhook → check logs (and optional GET delivery-status or DB)**. If logs show no errors and the webhook returned 200, the update path is working.

**On the server:** `docker compose logs -f orderb2bapi getdeliverystatus` (or `docker compose logs --tail=50 orderb2bapi`) to see recent lines after you send a webhook.

---

## When Wassel sends a payload (like WASSEL-WEBHOOK-SPEC.json) – how to check it worked

Wassel will POST to `https://webhooks.jafarshop.com/webhooks/wassel/status` with a body like:

```json
{"ItemReferenceNo":"1039","Status":170,"Waybill":"26020021000001","DeliveryImageUrl":"https://example.com/proof.jpg"}
```

(or minimal: `{"ItemReferenceNo":"1039","Status":130}`).

**1. Wassel side**  
- They should get **HTTP 200** and body `{"ok":true}`. If they get 401/503/400, the webhook isn’t accepted (check secret, URL, body).

**2. Our logs (server)**  
- **GetDeliveryStatus:** request to `POST /webhooks/wassel/status` → 200 (no error in logs).  
- **OrderB2bAPI:** a few seconds later you should see a line like  
  `"method":"POST","path":"/internal/webhooks/delivery","status":200`  
  That means GetDeliveryStatus forwarded the payload and we updated the order.

**3. API check**  
- Call GET delivery-status for the same reference Wassel used (`ItemReferenceNo` = e.g. `1039` or `ORDER-100`):  
  `GET https://api.jafarshop.com/v1/orders/1039/delivery-status` (with partner Bearer).  
- Response should include `"source": "stored"` and `shipment.status` / `shipment.status_label` matching what Wassel sent (e.g. 170 = "Delivered to customer").

**4. DB check (optional)**  
```sql
SELECT partner_order_id, shopify_order_id, last_delivery_status, last_delivery_status_label, last_delivery_waybill, last_delivery_at
FROM supplier_orders
WHERE shopify_order_id = '1039' OR partner_order_id = 'ORDER-100';
```
- Rows for that order should have `last_delivery_status`, `last_delivery_at` (and waybill/URL if Wassel sent them).

**Summary:** 200 from us to Wassel + `POST /internal/webhooks/delivery` 200 in OrderB2bAPI logs = it’s working. Use GET delivery-status or DB to confirm the stored values.

---

## All ways to check we received the webhook

Use any of these to confirm a webhook Wassel already sent was received and processed.

| # | Where | What to check |
|---|--------|----------------|
| 1 | **Wassel** | They got **HTTP 200** and body `{"ok":true}` from `POST https://webhooks.jafarshop.com/webhooks/wassel/status`. If not, we never accepted it (wrong URL, secret, or body). |
| 2 | **GetDeliveryStatus logs** | On the server: `docker compose logs getdeliverystatus` (or `--tail=100`). Look for: `Wassel webhook received: ItemReferenceNo=<ref> Status=<code> (<label>)` e.g. `ItemReferenceNo=1039 Status=170 (Delivered to customer)`. |
| 3 | **OrderB2bAPI logs** | On the server: `docker compose logs orderb2bapi`. Look for a request line: `"method":"POST","path":"/internal/webhooks/delivery","status":200` — that means GetDeliveryStatus forwarded the payload and we processed it. (If order not found you'll see an Info line with `no order for ItemReferenceNo`; if lookup failed, a Warn.) |
| 4 | **GET delivery-status API** | Call with partner auth: `GET https://api.jafarshop.com/v1/orders/<id>/delivery-status` (use the same id Wassel used as `ItemReferenceNo`, e.g. `1039` or order UUID). Response should have `"source": "stored"` and `shipment.status` / `shipment.status_label` / `shipment.updated_at` matching the webhook. |
| 5 | **Database** | Run on the server: `docker compose exec postgres psql -U postgres -d b2bapi -c "SELECT partner_order_id, shopify_order_id, last_delivery_status, last_delivery_status_label, last_delivery_at FROM supplier_orders WHERE last_delivery_at IS NOT NULL ORDER BY last_delivery_at DESC LIMIT 20;"` — or filter by `shopify_order_id = '1039'` / `partner_order_id = 'ORDER-100'`. You should see `last_delivery_status`, `last_delivery_status_label`, and `last_delivery_at` set. |

**Short version:** If Wassel got 200 and you see `POST /internal/webhooks/delivery` 200 in OrderB2bAPI logs, we received it. Use GET delivery-status or the DB to see the stored status.

---

## Auto-create from Wassel (when `WASSEL_DEFAULT_PARTNER_ID` is set)

When **OrderB2bAPI** has `WASSEL_DEFAULT_PARTNER_ID` set (UUID of a partner), any webhook for an `ItemReferenceNo` that does not exist in the database will:

1. Create a minimal supplier order under that partner (`partner_order_id` and, if numeric, `shopify_order_id` = ItemReferenceNo; status UNFULFILLED; customer "Wassel delivery").
2. Store the delivery status on that order (`last_delivery_*`).
3. Respond 200 to Wassel as usual.

So **any order registered at Wassel** ends up in our database. **Partners see only their own orders** (enforced by `partner_id` on all APIs): the default partner sees these Wassel-created orders; others see only orders they created. If the same order is later created by the real partner (e.g. via cart), future webhooks for that reference update the real partner’s order (we prefer non–default-partner when multiple rows share the same Shopify order number).

---

## Checklist (Wassel ↔ you only)

| Step | What | Expected |
|------|------|----------|
| 1 | POST internal webhook (OrderB2bAPI) with `ItemReferenceNo` + `Status` 170 | 200, `ok: true`, order found |
| 2 | POST to GetDeliveryStatus (Wassel URL) with same payload + Bearer `WASSEL_SHARED_SECRET` | 200, `ok: true` |
| 3 | **Check logs** (OrderB2bAPI + GetDeliveryStatus) | No errors; webhook 200 and (if applicable) forward success |
| 4 | (Optional) GET delivery-status or SELECT in DB | `source: "stored"` or `last_delivery_*` filled |

Once this works, you can later test **partner notification** (set `webhook_url` on the partner and send a delivery webhook again; partner should receive a POST).

---

## Status codes (Wassel)

| Code | Meaning |
|------|--------|
| 170 | Delivered to customer |
| 130 | Out for delivery |
| 100 | Picked up by driver |
| 51–58, 60, 120, 121, 123 | In transit |
| 180, 190, 210 | Returned |

See `GetDeliveryStatus/WASSEL-WEBHOOK-SPEC.json` for the full list.

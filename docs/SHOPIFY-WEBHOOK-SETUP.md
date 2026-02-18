# Shopify Webhook Setup (auto-sync order status)

This webhook updates our `supplier_orders.status` automatically when an order is fulfilled in Shopify.

---

## 1. Configure the secret on the server

In the **server** `.env` (repo root, same folder as `docker-compose.yml`), set:

```
SHOPIFY_WEBHOOK_SECRET=your_webhook_secret_from_shopify
```

Restart `orderb2bapi` after setting it:

```bash
docker compose up -d --force-recreate orderb2bapi
```

---

## 2. Create the webhook in Shopify Admin

In Shopify Admin:

- Go to **Settings → Notifications → Webhooks**
- Click **Create webhook**

Create webhooks for these topics:

- **`fulfillments/create`**
- **`fulfillments/update`**

Use this URL (production):

```
https://api.jafarshop.com/webhooks/shopify/fulfillment
```

Format: **JSON**

When Shopify asks for a secret, use the same value you put in `SHOPIFY_WEBHOOK_SECRET`.

---

## 3. What happens

When Shopify sends a fulfillment webhook:

- We verify `X-Shopify-Hmac-Sha256`.
- We find our order using Shopify order name (e.g. `#1040` → `1040`) in `supplier_orders.shopify_order_id`.
- We update:
  - `supplier_orders.status` → `FULFILLED`
  - tracking fields if Shopify included them (`tracking_number`, `tracking_company`, `tracking_url`)

---

## 4. Quick test

After fulfilling an order in Shopify, check DB:

```sql
SELECT partner_order_id, shopify_order_id, status, tracking_number
FROM supplier_orders
ORDER BY created_at DESC
LIMIT 10;
```


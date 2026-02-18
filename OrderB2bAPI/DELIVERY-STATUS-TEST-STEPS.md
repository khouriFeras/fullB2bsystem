# Test: Store delivery status from webhook & GET returns stored

## Step 1 – On the server (e.g. ubuntu)

```bash
cd /home/feras/b2b
git pull
```

## Step 2 – Run migration (adds last_delivery_* columns)

From repo root `b2b/`:

```bash
./deploy/run-migrations.sh
```

Or if DB is in Docker and you use host port 5434:

```bash
docker run --rm -v "$(pwd)/OrderB2bAPI/migrations:/migrations" --network host \
  migrate/migrate -path /migrations \
  -database "postgres://postgres:YOUR_DB_PASSWORD@127.0.0.1:5434/b2bapi?sslmode=disable" up
```

## Step 3 – Rebuild and restart API

```bash
docker compose build orderb2bapi
docker compose up -d orderb2bapi
```

## Step 4 – Send a delivery webhook (store status)

Use an order that exists (e.g. shopify_order_id or partner_order_id like `1039`):

```bash
curl -s -X POST "https://api.jafarshop.com/internal/webhooks/delivery" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_DELIVERY_WEBHOOK_SECRET" \
  -d '{"ItemReferenceNo":"1039","Status":170,"Waybill":"WB123"}'
```

Expected: `"status":"no_webhook"` or `"status":"forwarded"` and `"ok":true`. The order’s last delivery status is now stored.

## Step 5 – GET delivery-status (return stored)

With partner auth, call GET for that order (use the order’s `partner_order_id` or order UUID as `:id`):

```bash
curl -s -H "Authorization: Bearer YOUR_PARTNER_API_KEY" \
  "https://api.jafarshop.com/v1/orders/PARTNER_ORDER_ID_OR_UUID/delivery-status"
```

Expected: JSON with `"source":"stored"` and `shipment` containing `status`, `status_label`, `waybill`, `delivery_image_url`, `updated_at` from the last webhook.

If no webhook was received yet for that order, the API falls back to calling GetDeliveryStatus (Wassel) as before.

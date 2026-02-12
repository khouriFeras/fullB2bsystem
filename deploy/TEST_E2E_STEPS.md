# Step-by-step E2E test (see every server response)

Run these on your **server** (SSH into `feras@95.217.6.87`), from the `~/b2b` directory. Each command prints the **server response** in the terminal.

Replace:
- `YOUR_COLLECTION_HANDLE` with a real Shopify collection handle (e.g. `wholesale`)
- After Step 1, replace `YOUR_API_KEY` with the API key printed by create-partner
- After Step 2, replace `SOME-SKU` with a real `sku` from the catalog JSON (if you have products)

---

## Step 0: Health check

```bash
cd ~/b2b
curl -s http://127.0.0.1:8081/health
```

**You should see:** `{"status":"ok"}` (that’s the server response in the terminal).

---

## Step 1: Create a partner

```bash
docker compose run --rm --entrypoint ./create-partner orderb2bapi "E2E Test Partner" "my-test-api-key-12345" "YOUR_COLLECTION_HANDLE"
```

**You should see:** A block of text including “Partner created successfully!”, “API Key: my-test-api-key-12345”, and “Authorization: Bearer my-test-api-key-12345”. **Copy that API key** and use it as `YOUR_API_KEY` in the next steps (e.g. `my-test-api-key-12345`).

---

## Step 2: Catalog (partner receives products)

Set your API key once (use the key from Step 1):

```bash
export KEY="my-test-api-key-12345"
export API="http://127.0.0.1:8081"
```

Then request the catalog:

```bash
curl -s -H "Authorization: Bearer $KEY" "$API/v1/catalog/products?limit=10"
```

**You should see:** JSON in the terminal, e.g. `{"data":[...],"pagination":{...}}`. Each item in `data` has at least `sku`, `title`, `price`. **Pick one `sku`** from the list (e.g. `ABC123`) for the next step. If `data` is empty, fix the collection handle or Shopify/ProductB2B config, then retry.

---

## Step 3: Create an order (cart submit)

Replace `SOME-SKU` with a real SKU from Step 2. Use a unique idempotency key each time:

```bash
curl -s -X POST "$API/v1/carts/submit" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: step3-$(date +%s)" \
  -d '{
    "partner_order_id": "E2E-ORDER-001",
    "items": [{"sku": "SOME-SKU", "title": "Test Product", "price": 10, "quantity": 1}],
    "customer": {"first_name": "Test", "last_name": "User", "phone_number": "+962700000000", "email": "e2e@test.com"},
    "shipping": {"city": "Amman", "address": "123 St", "postal_code": "11118", "country": "JO"},
    "totals": {"subtotal": 10, "tax": 0, "shipping": 0, "total": 10}
  }'
```

**You should see:** Either JSON like `{"supplier_order_id":"...","status":"PENDING_CONFIRMATION",...}` (order created) or nothing (HTTP 204 = no JafarShop items matched; use a SKU from the catalog).

---

## Step 4: Get the order

```bash
curl -s -H "Authorization: Bearer $KEY" "$API/v1/orders/E2E-ORDER-001"
```

**You should see:** JSON with order details (id, status, items, shipping, etc.). That’s the server response in the terminal.

---

## Step 5: Delivery status

```bash
curl -s -H "Authorization: Bearer $KEY" "$API/v1/orders/E2E-ORDER-001/delivery-status"
```

**You should see:** Either JSON with `shipment` (and maybe Wassel data) or an error object (e.g. 502 if GetDeliveryStatus/Wassel is unavailable or the reference isn’t found). The full response is in the terminal.

---

## Optional: List orders

```bash
curl -s -H "Authorization: Bearer $KEY" "$API/v1/admin/orders?limit=5"
```

**You should see:** JSON list of orders for this partner.

---

## Run the script and still see responses

To run the full flow in one go **and** print each server response in the terminal:

```bash
cd ~/b2b
sed -i 's/\r$//' deploy/test-e2e.sh
chmod +x deploy/test-e2e.sh
SHOW_RESPONSES=1 COLLECTION_HANDLE=YOUR_COLLECTION_HANDLE ./deploy/test-e2e.sh
```

With an existing API key:

```bash
SHOW_RESPONSES=1 API_BASE=http://127.0.0.1:8081 PARTNER_API_KEY=my-test-api-key-12345 ./deploy/test-e2e.sh
```

Responses appear after each step under lines like `--- Catalog response ---`.

# How partners reach the API and receive updates

Once DNS and Nginx are in place, partners use the **public B2B API** over HTTPS. Until then they can use the server IP for testing.

---

## 1. How partners reach the API

| Item | Value |
|------|--------|
| **Base URL (production)** | `https://api.jafarshop.com` |
| **Base URL (testing)** | `http://95.217.6.87:8081` |
| **Authentication** | HTTP header: `Authorization: Bearer <partner_api_key>` |

You give each partner:

- Their **API key** (the secret you set when you ran `create-partner`).
- The **base URL** (production or test).

They send the API key on every request in the header above. No login endpoint; the key itself is the credential.

**Main endpoints:**

| Method | Path | Purpose |
|--------|------|--------|
| GET | `/health` | Check API is up (no auth) |
| GET | `/v1/catalog/products?limit=...` | List products for their collection |
| POST | `/v1/carts/submit` | Submit cart → create order |
| GET | `/v1/orders/:partner_order_id` | Get order details |
| GET | `/v1/orders/:partner_order_id/delivery-status` | Get delivery/shipment status |

Example (curl):

```bash
curl -s -H "Authorization: Bearer PARTNER_API_KEY" \
  "https://api.jafarshop.com/v1/catalog/products?limit=10"
```

---

## 2. How partners get updates

### Catalog / product updates

- **No push.** Partners get the latest catalog by **calling the API**:  
  `GET /v1/catalog/products`.
- ProductB2B is kept in sync with Shopify (via Shopify → ProductB2B webhooks). Each catalog call returns current data for that partner’s collection.

So partners “receive” catalog updates by **polling** the catalog endpoint when they need fresh data (e.g. before building a cart or refreshing their UI).

### Order and delivery updates

- **Option A – Polling**  
  Partners call:
  - `GET /v1/orders/:partner_order_id` for order status
  - `GET /v1/orders/:partner_order_id/delivery-status` for shipment/delivery info  
  whenever they need an update (e.g. on a timer or when the user opens “my orders”).

- **Option B – Webhook (push)**  
  You can set a **partner webhook URL** so your system **POSTs delivery updates** to the partner’s server when delivery status is fetched or updated.

  **On the server** (from `~/b2b`), after the partner exists.  
  *(Note: The Docker image currently only includes `create-partner`. To run `list-partners` or `set-partner-webhook` on the server you would need to add those binaries to the image, or run them from a machine that has the OrderB2bAPI repo and can connect to the server DB.)*

  ```bash
  # List partners to get partner ID (if binary available in image)
  docker compose run --rm --entrypoint ./list-partners orderb2bapi

  # Set partner webhook URL (partner receives POST when delivery status is available)
  docker compose run --rm --entrypoint ./set-partner-webhook orderb2bapi \
    --partner-id <PARTNER_UUID> \
    --webhook-url "https://partner-their-server.com/webhooks/jafarshop-delivery"
  ```

  Alternatively, from your **PC** (with the repo and DB connection to the server):  
  `go run cmd/list-partners/main.go` and `go run cmd/set-partner-webhook/main.go --partner-id <uuid> --webhook-url <url>`.

  To clear the webhook:

  ```bash
  docker compose run --rm --entrypoint ./set-partner-webhook orderb2bapi \
    --partner-id <PARTNER_UUID> --webhook-url ""
  ```

  **Webhook payload** (JSON POST to their URL):

  - `partner_id`, `order_id`, `partner_order_id`
  - `shipping_address`
  - `shipment` (from GetDeliveryStatus/Wassel)
  - `event`: e.g. `"delivery_status"`

  Their endpoint should respond with **2xx** so your side logs it as sent successfully. Sending is fire-and-forget (async).

---

## 3. Summary

| What | How partners get it |
|------|----------------------|
| **Reach the API** | Use base URL + `Authorization: Bearer <api_key>` on every request. |
| **Catalog updates** | Call `GET /v1/catalog/products` when they need fresh products (polling). |
| **Order/delivery updates** | Poll `GET /v1/orders/:id` and `GET /v1/orders/:id/delivery-status`, **or** you set a webhook URL so they receive delivery updates via POST to their server. |

You create partners with `create-partner` (name, API key, collection handle). You optionally set a webhook URL with `set-partner-webhook` so they receive delivery updates without polling.

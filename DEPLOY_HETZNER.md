# Deploy B2B to Hetzner CX43

This guide walks through deploying the full B2B stack (PostgreSQL, OrderB2bAPI, ProductB2B, GetDeliveryStatus) on a Hetzner CX43 server using the root Docker Compose.

## Stack overview

| Service           | Host port | Purpose                      |
| ----------------- | --------- | ---------------------------- |
| postgres          | 5434      | Database                     |
| productb2b        | 3000      | Product/catalog API (Go)     |
| getdeliverystatus | 5000      | Delivery webhook API (Flask) |
| orderb2bapi       | 8081      | Main B2B API (Go)            |

---

## 1. Server preparation (Hetzner CX43)

- **OS**: Ubuntu 22.04 or 24.04 LTS.
- **One-time setup**: Run the provided script (as root or with sudo):

  ```bash
  sudo bash deploy/setup-server.sh
  ```

  This installs Docker, the Docker Compose plugin, and configures UFW (SSH 22, HTTP 80, HTTPS 443). Log out and back in (or run `newgrp docker`) so your user can run Docker without sudo.

- **Optional**: Create a dedicated user and app directory, e.g. `/home/deploy/b2b`.

---

## 2. Deploy the application

### 2.1 Get code on the server

Clone the repo or copy the project (e.g. `rsync`/`scp`) into your app directory (e.g. `/home/deploy/b2b`).

### 2.2 Root `.env`

Copy the example and fill in real values:

```bash
cp .env.example .env
# Edit .env: DB_PASSWORD, DB_NAME, SHOPIFY_SHOP_DOMAIN, SHOPIFY_ACCESS_TOKEN,
# PRODUCT_B2B_SERVICE_API_KEY, API_KEY_HASH_SALT (use a strong random value in production).
```

See [.env.example](.env.example) for all variables. Compose overrides `DB_HOST`, `PORT`, `PRODUCT_B2B_URL`, and `GET_DELIVERY_STATUS_URL` for the orderb2bapi container.

### 2.3 GetDeliveryStatus `.env`

```bash
cp GetDeliveryStatus/.env.example GetDeliveryStatus/.env
# Fill in Wassel credentials (STORE_MODE, test/actual store vars). See GetDeliveryStatus/.env.example.
```

### 2.4 ProductB2B `.shopify_token`

The compose file mounts `./ProductB2B/.shopify_token`. Either:

- Create the file and paste your Shopify admin token into it, or  
- Use ProductB2B’s OAuth flow once so it writes the token into the file.

### 2.5 Start the stack

From the repo root (`b2b/`):

```bash
docker compose up -d
docker compose ps
docker compose logs -f
```

Fix any startup errors before continuing.

---

## 3. Run database migrations (one-time)

OrderB2bAPI does not run migrations automatically. After the first `docker compose up -d`, run:

```bash
# From repo root; uses .env for DB_PASSWORD and DB_NAME
chmod +x deploy/run-migrations.sh
./deploy/run-migrations.sh
```

Or manually with the migrate image (replace `YOUR_DB_PASSWORD` and `YOUR_DB_NAME` to match your `.env`):

```bash
docker run --rm -v "$(pwd)/OrderB2bAPI/migrations:/migrations" --network host \
  migrate/migrate -path /migrations \
  -database "postgres://postgres:YOUR_DB_PASSWORD@127.0.0.1:5434/YOUR_DB_NAME?sslmode=disable" up
```

Then restart the API if needed:

```bash
docker compose restart orderb2bapi
```

---

## 4. Optional: Reverse proxy and SSL (production)

For production, put Nginx (or Caddy) in front and use HTTPS. GetDeliveryStatus and Shopify webhooks must be served over HTTPS.

- Install Nginx and Certbot (e.g. `sudo apt install nginx certbot python3-certbot-nginx`).
- Use the sample configs in `deploy/` (preconfigured for **jafarshop.com** subdomains):
  - [deploy/nginx-api.conf](deploy/nginx-api.conf) – OrderB2bAPI → `https://api.jafarshop.com` → `http://127.0.0.1:8081`
  - [deploy/nginx-products.conf](deploy/nginx-products.conf) – ProductB2B → `https://products.jafarshop.com` → `http://127.0.0.1:3000`
  - [deploy/nginx-webhooks.conf](deploy/nginx-webhooks.conf) – GetDeliveryStatus → `https://webhooks.jafarshop.com` → `http://127.0.0.1:5000`

Copy each file to `/etc/nginx/sites-available/`, enable the site, run Certbot for the hostnames, then `sudo nginx -t && sudo systemctl reload nginx`. With a reverse proxy, you can leave 8081, 3000, and 5000 closed to the internet and only expose 80/443.

**Full steps for jafarshop.com (DNS, Nginx, SSL, Shopify App URL, webhooks):** see [deploy/JAFARSHOP_DOMAIN.md](deploy/JAFARSHOP_DOMAIN.md).

---

## 5. Post-deploy checks

- **Health**:  
  - OrderB2bAPI: `curl http://localhost:8081/health` (or your Nginx URL).  
  - GetDeliveryStatus: `curl http://localhost:5000/health`.
- **Logs**: `docker compose logs -f orderb2bapi productb2b getdeliverystatus`.
- **Partners/DB**: Use OrderB2bAPI’s `create-partner` (or scripts) against the deployed API. From the server host use `DB_HOST=127.0.0.1` and `DB_PORT=5434` to target the same Postgres.

---

## 6. E2E test (optional)

To run the full flow (create partner → catalog → cart submit → order get → delivery status), use the E2E test script.

**On the server** (from `~/b2b`), with a real Shopify collection handle that has products:

```bash
sed -i 's/\r$//' deploy/test-e2e.sh   # fix CRLF if copied from Windows
COLLECTION_HANDLE=wholesale bash deploy/test-e2e.sh
```

Replace `wholesale` with your Shopify collection handle. The script will create a partner, then call catalog, submit a cart (using the first SKU from the catalog), fetch the order, and call delivery-status. If `./deploy/test-e2e.sh` says "No such file or directory", use `bash deploy/test-e2e.sh` instead.

**Using an existing partner API key** (from your PC or server):

```bash
API_BASE=http://95.217.6.87:8081 PARTNER_API_KEY=your-saved-api-key bash deploy/test-e2e.sh
```

See [deploy/test-e2e.sh](deploy/test-e2e.sh) for env vars (`API_BASE`, `PARTNER_API_KEY`, `COLLECTION_HANDLE`). To **see every server response** in the terminal, run with `SHOW_RESPONSES=1` or follow the manual steps in [deploy/TEST_E2E_STEPS.md](deploy/TEST_E2E_STEPS.md).

---

## Files added for deployment

| File | Purpose |
| ---- | ------- |
| [.env.example](.env.example) | Template for root `.env` (postgres, OrderB2bAPI, ProductB2B vars). |
| [deploy/setup-server.sh](deploy/setup-server.sh) | One-time server setup: Docker, Compose, UFW. |
| [deploy/run-migrations.sh](deploy/run-migrations.sh) | One-time DB migrations using golang-migrate. |
| [deploy/test-e2e.sh](deploy/test-e2e.sh) | E2E test: partner (optional), catalog, cart submit, order get, delivery status. |
| [deploy/nginx-api.conf](deploy/nginx-api.conf) | Nginx sample for OrderB2bAPI (HTTPS, api.jafarshop.com). |
| [deploy/nginx-products.conf](deploy/nginx-products.conf) | Nginx sample for ProductB2B (HTTPS, products.jafarshop.com). |
| [deploy/nginx-webhooks.conf](deploy/nginx-webhooks.conf) | Nginx sample for GetDeliveryStatus (HTTPS, webhooks.jafarshop.com). |
| [deploy/JAFARSHOP_DOMAIN.md](deploy/JAFARSHOP_DOMAIN.md) | Step-by-step: connect stack to jafarshop.com (DNS, Nginx, SSL, Shopify). |
| [deploy/PARTNER_API_ACCESS.md](deploy/PARTNER_API_ACCESS.md) | How partners reach the API and receive updates (URL, auth, polling, webhook). |

No code changes are required for a basic deploy; only server setup, env files, and one-time migrations.

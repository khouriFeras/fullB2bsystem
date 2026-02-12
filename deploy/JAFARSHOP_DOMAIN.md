# Connect B2B stack to jafarshop.com

Use subdomains of [jafarshop.com](https://www.jafarshop.com/) so the B2B API, ProductB2B, and delivery webhooks are served over HTTPS. Your main store stays at **www.jafarshop.com**; these are separate subdomains pointing to your Hetzner server.

---

## Test now (without DNS)

You can test everything using the server IP and ports. No DNS needed.

**From your PC (PowerShell):**

```powershell
# Health
Invoke-RestMethod -Uri "http://95.217.6.87:8081/health"

# Catalog (use your real partner API key)
$key = "partner123"
Invoke-RestMethod -Uri "http://95.217.6.87:8081/v1/catalog/products?limit=10" -Headers @{ Authorization = "Bearer $key" }
```

**From the server** (SSH into `feras@95.217.6.87`, then `cd ~/b2b`):

```bash
# Health
curl -s http://127.0.0.1:8081/health

# Full E2E (create partner + catalog + cart + order + delivery status)
COLLECTION_HANDLE=partner-zain bash deploy/test-e2e.sh

# Or use existing partner key (from PC or server)
API_BASE=http://95.217.6.87:8081 PARTNER_API_KEY=partner123 bash deploy/test-e2e.sh
```

If you get "No such file or directory" when running the script: (1) ensure the file exists (`ls deploy/test-e2e.sh`); (2) if the file is missing on the server, copy the `deploy` folder from your PC (see below); (3) run with `bash deploy/test-e2e.sh` instead of `./deploy/test-e2e.sh`; (4) if it was copied from Windows, fix line endings: `sed -i 's/\r$//' deploy/test-e2e.sh` then try again.

**Copy deploy files from your PC to the server** (run from your PC in the repo root `b2b/`):

- **Git Bash** (if you have Git for Windows): open "Git Bash", then:
  ```bash
  cd /d/JafarShop/b2b
  scp -r deploy feras@95.217.6.87:~/b2b/
  ```
- **PowerShell and OpenSSH**: If `scp` is not recognized, try the full path (OpenSSH is often installed but not in PATH):
  ```powershell
  & "C:\Windows\System32\OpenSSH\scp.exe" -r deploy feras@95.217.6.87:~/b2b/
  ```
  If that fails, add OpenSSH Client: **Settings → Apps → Optional features → Add a feature → OpenSSH Client**, then open a new PowerShell and try `scp` again.
- **WinSCP** (GUI): Download [WinSCP](https://winscp.net/), connect to `feras@95.217.6.87`, navigate to `~/b2b/`, and drag the `deploy` folder from your PC into it.

Then on the server: `cd ~/b2b` and run `COLLECTION_HANDLE=partner-zain bash deploy/test-e2e.sh`.

Replace `partner123` and `partner-zain` with your real API key and collection handle. See [deploy/TEST_E2E_STEPS.md](TEST_E2E_STEPS.md) for manual step-by-step with curl.

---

## Test api.jafarshop.com (after DNS + Nginx)

**1. Check DNS** (PowerShell on your PC):

```powershell
Resolve-DnsName api.jafarshop.com -Type A
```

You should see `95.217.6.87`. If not, add the A record and wait a few minutes.

**2. Call the API via the domain** (only works after Nginx is set up on the server):

```powershell
# HTTP (works after Nginx is configured and listening on port 80)
Invoke-RestMethod -Uri "http://api.jafarshop.com/health"

# HTTPS (works after Certbot has issued a certificate for api.jafarshop.com)
Invoke-RestMethod -Uri "https://api.jafarshop.com/health"
```

If you get "connection refused" or timeout: ensure Nginx is installed, the config is in place, and port 80 (and 443 for HTTPS) is open. See sections 1–2 below.

---

## Add products.jafarshop.com and webhooks.jafarshop.com

Use the same flow as api: DNS → Nginx (HTTP first) → Certbot. Server already has Nginx and Certbot; default site is removed.

### Step 1: DNS (Cloudflare or your registrar)

Add A records; use **DNS only** (grey cloud) if on Cloudflare:

| Type | Name     | Value       |
|------|----------|-------------|
| A    | products | 95.217.6.87 |
| A    | webhooks | 95.217.6.87 |

### Step 2: Nginx configs on the server

SSH in, then run these (creates files directly – no need for deploy folder):

```bash
# ProductB2B (port 3000)
sudo tee /etc/nginx/sites-available/b2b-products << 'EOF'
server {
    listen 80;
    server_name products.jafarshop.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

# GetDeliveryStatus (port 5000)
sudo tee /etc/nginx/sites-available/b2b-webhooks << 'EOF'
server {
    listen 80;
    server_name webhooks.jafarshop.com;

    location / {
        proxy_pass http://127.0.0.1:5000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

# Enable and reload
sudo ln -sf /etc/nginx/sites-available/b2b-products /etc/nginx/sites-enabled/
sudo ln -sf /etc/nginx/sites-available/b2b-webhooks /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Step 3: SSL with Certbot

```bash
sudo certbot --nginx -d products.jafarshop.com -d webhooks.jafarshop.com --non-interactive --agree-tos -m feras.khouri@jafarshop.com
```

(Or run without `--non-interactive` if you prefer to answer prompts.)

### Step 4: ProductB2B – set PUBLIC_BASE_URL

```bash
cd ~/b2b
grep -q "PUBLIC_BASE_URL" .env || echo "PUBLIC_BASE_URL=https://products.jafarshop.com" >> .env
# If the line exists with wrong value, edit .env and set: PUBLIC_BASE_URL=https://products.jafarshop.com
docker compose restart productb2b
```

### Step 5: Verify

From your PC:

```powershell
Invoke-RestMethod -Uri "https://products.jafarshop.com/health"
Invoke-RestMethod -Uri "https://webhooks.jafarshop.com/health"
```

---

## 1. DNS (where jafarshop.com is managed)

Add **A** records pointing to your server IP **95.217.6.87**:

| Type | Name  | Value       | TTL (optional) |
|------|-------|-------------|----------------|
| A    | api   | 95.217.6.87 | 300            |
| A    | products | 95.217.6.87 | 300         |
| A    | webhooks | 95.217.6.87 | 300         |

Result:

- **api.jafarshop.com** → 95.217.6.87 (OrderB2bAPI)
- **products.jafarshop.com** → 95.217.6.87 (ProductB2B, Shopify app + webhooks)
- **webhooks.jafarshop.com** → 95.217.6.87 (GetDeliveryStatus)

Wait until DNS propagates (e.g. `nslookup api.jafarshop.com` returns 95.217.6.87).

---

## 2. Server: Nginx + SSL

**On the server** (SSH as feras, then sudo where needed):

```bash
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx
```

Copy the Nginx configs from the repo (from your machine or after pulling on the server):

```bash
cd ~/b2b
sudo cp deploy/nginx-api.conf /etc/nginx/sites-available/b2b-api
sudo cp deploy/nginx-products.conf /etc/nginx/sites-available/b2b-products
sudo cp deploy/nginx-webhooks.conf /etc/nginx/sites-available/b2b-webhooks
sudo ln -s /etc/nginx/sites-available/b2b-api /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/b2b-products /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/b2b-webhooks /etc/nginx/sites-enabled/
```

Get certificates (run once DNS is correct):

```bash
sudo certbot --nginx -d api.jafarshop.com -d products.jafarshop.com -d webhooks.jafarshop.com --non-interactive --agree-tos -m YOUR_EMAIL@example.com
```

Replace `YOUR_EMAIL@example.com` with your email for Let's Encrypt.

Test and reload Nginx:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

---

## 3. Server: set PUBLIC_BASE_URL for ProductB2B

ProductB2B needs the public HTTPS URL for registering Shopify webhooks:

```bash
cd ~/b2b
grep -q "PUBLIC_BASE_URL" .env || echo "PUBLIC_BASE_URL=https://products.jafarshop.com" >> .env
```

If you use nano or another editor, set:

```env
PUBLIC_BASE_URL=https://products.jafarshop.com
```

Restart the stack so ProductB2B picks it up:

```bash
docker compose down
docker compose up -d
```

---

## 4. Shopify: App URL and webhooks

- In **Shopify Partner Dashboard** → your app → **App setup** (or **Configuration**):
  - **App URL:** `https://products.jafarshop.com`
  - **Allowed redirection URL(s):** add `https://products.jafarshop.com/auth/callback` (or the path your OAuth uses) if needed.

- **Register webhooks** with ProductB2B (so Shopify sends product/inventory updates to your server):

  Set **ADMIN_SETUP_KEY** in `~/b2b/.env` (e.g. a random string), then call:

  ```bash
  curl -X POST https://products.jafarshop.com/admin/setup/webhooks \
    -H "X-Setup-Key: YOUR_ADMIN_SETUP_KEY"
  ```

  Or from PowerShell:

  ```powershell
  Invoke-RestMethod -Uri "https://products.jafarshop.com/admin/setup/webhooks" -Method POST -Headers @{ "X-Setup-Key" = "YOUR_ADMIN_SETUP_KEY" }
  ```

  Response should show `ok: true` for each webhook topic. Then Shopify will send product/inventory webhooks to `https://products.jafarshop.com/webhooks/...`.

---

## 5. Use the new URLs

| Service           | URL                              |
|-------------------|-----------------------------------|
| B2B API (catalog, orders, health) | https://api.jafarshop.com        |
| ProductB2B (catalog, webhooks)    | https://products.jafarshop.com   |
| GetDeliveryStatus (delivery webhooks) | https://webhooks.jafarshop.com |

Examples:

- Health: `https://api.jafarshop.com/health`
- Catalog: `GET https://api.jafarshop.com/v1/catalog/products?limit=10` with `Authorization: Bearer <partner_api_key>`

---

## Summary

1. **DNS:** A records for **api**, **products**, **webhooks**.jafarshop.com → 95.217.6.87  
2. **Server:** Nginx + Certbot; copy `deploy/nginx-*.conf`, run certbot for the three subdomains.  
3. **.env:** `PUBLIC_BASE_URL=https://products.jafarshop.com`; restart stack.  
4. **Shopify:** App URL = `https://products.jafarshop.com`; call `/admin/setup/webhooks` to register webhooks.

Your store stays at https://www.jafarshop.com/; the B2B stack is on the subdomains above.

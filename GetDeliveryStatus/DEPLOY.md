# GetDeliveryStatus – Deployment (Hostinger and Docker)

This app exposes a small Flask API (`/health`, `/shipment`) used as a delivery-status webhook service. Deploy it on **Hostinger VPS** (Gunicorn + Nginx) or **Hostinger Docker**.

---

## Requirements

- **Python 3.10+** (or use the provided Dockerfile)
- Environment variables: Wassel API credentials (see [Environment variables](#environment-variables))

---

## 1. Hostinger VPS (Gunicorn + Nginx)

### 1.1 Install dependencies

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install python3 python3-pip python3-venv nginx -y
```

### 1.2 App directory and venv

```bash
mkdir -p /home/youruser/getdeliverystatus
cd /home/youruser/getdeliverystatus
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

(If you need a minimal image and **do not** use the optional Playwright feature, you can use a `requirements.txt` that omits the `playwright` line to keep the environment smaller.)

### 1.3 Run with Gunicorn (configurable port)

```bash
export PORT=5000
gunicorn --bind 0.0.0.0:${PORT:-5000} --workers 2 --threads 2 --timeout 60 wsgi:app
```

### 1.4 Systemd service

A sample unit is in `deploy/getdeliverystatus.service`. Create `/etc/systemd/system/getdeliverystatus.service` (adjust `youruser` and paths):

```ini
[Unit]
Description=Gunicorn instance for GetDeliveryStatus
After=network.target

[Service]
User=youruser
Group=www-data
WorkingDirectory=/home/youruser/getdeliverystatus
Environment="PATH=/home/youruser/getdeliverystatus/venv/bin"
Environment="PORT=5000"
ExecStart=/home/youruser/getdeliverystatus/venv/bin/gunicorn --workers 2 --threads 2 --timeout 60 --bind unix:getdeliverystatus.sock -m 007 wsgi:app

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl start getdeliverystatus
sudo systemctl enable getdeliverystatus
sudo systemctl status getdeliverystatus
```

### 1.5 Nginx (HTTPS required for webhooks)

**Webhooks must be served over HTTPS.** Use Let’s Encrypt (e.g. Certbot) and configure Nginx for port 443.

A sample config is in `deploy/nginx-getdeliverystatus.conf`. Example server block (after SSL is set up). Replace `your_domain` and the socket path:

```nginx
server {
    listen 443 ssl;
    server_name your_domain www.your_domain;

    # SSL managed by Certbot / Let's Encrypt
    # ssl_certificate /etc/letsencrypt/live/your_domain/fullchain.pem;
    # ssl_certificate_key /etc/letsencrypt/live/your_domain/privkey.pem;

    location / {
        include proxy_params;
        proxy_pass http://unix:/home/youruser/getdeliverystatus/getdeliverystatus.sock;
    }
}
```

Enable the site, test Nginx, reload:

```bash
sudo ln -s /etc/nginx/sites-available/getdeliverystatus /etc/nginx/sites-enabled
sudo nginx -t
sudo systemctl reload nginx
```

### 1.6 Firewall

```bash
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

---

## 2. Hostinger Docker

- Use the project **Dockerfile** (it runs Gunicorn with `--workers 2 --threads 2 --timeout 60` and bind `0.0.0.0:${PORT:-5000}`).
- In Hostinger Docker Manager, set **environment variables** (see below); do **not** bake secrets into the image. Mount or inject a `.env` file, or set each variable in the container config.
- Expose the container port (e.g. 5000) and put a reverse proxy (Nginx/Traefik) in front with **HTTPS** for webhook endpoints.

Example run:

```bash
docker build -t getdeliverystatus ./GetDeliveryStatus
docker run -d --name getdeliverystatus -p 5000:5000 -e PORT=5000 --env-file ./GetDeliveryStatus/.env getdeliverystatus
```

---

## 3. Environment variables

Set these in `.env` (VPS) or in the container / env file (Docker). See `.env.example` for a template.

| Variable | Required | Description |
|----------|----------|-------------|
| `STORE_MODE` | No (default `test`) | `test` or `actual` – which Wassel store to use |
| `TEST_Email`, `TEST_Password`, `TEST_CompanyId`, `TEST_StoreId`, `TEST_Token`, `TEST_Base_url` | Yes if `STORE_MODE=test` | Test store Wassel credentials |
| `ACTUAL_Email`, `ACTUAL_Password`, `ACTUAL_Token`, `ACTUAL_Base_url` | Yes if `STORE_MODE=actual` | Production store Wassel credentials |
| `PORT` | No (default `5000`) | Port Gunicorn binds to |
| `VERBOSE` | No | Set to `1` or `true` for extra logging |

**Playwright** is an optional dependency (in `requirements.txt`). The app runs without it; omit it from `requirements.txt` for a smaller install if you do not use that feature.

---

## 4. Health check

- **HTTP:** `GET /health` returns `{"status":"ok"}`.
- Use this for load balancers, Docker HEALTHCHECK, and monitoring.

---

## 5. Docker Compose (repo root)

The root `docker-compose.yml` includes a `getdeliverystatus` service. It uses `GetDeliveryStatus/.env` and maps port 5000. The Dockerfile uses `PORT` from the environment (default 5000), so the stack works as-is; you can set `environment: PORT: "5000"` on the service if you want it explicit.

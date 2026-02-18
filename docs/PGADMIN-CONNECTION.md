# Connect pgAdmin to the B2B database

---

## A. Connect to the **server** database (the one already in use)

This is the same Postgres the app uses in production (e.g. on the Ubuntu server). Use the **same credentials** as on the server (from the server’s `.env`).

### 1. What you need from the server

- **Host:** The server’s hostname or IP (e.g. `api.jafarshop.com` or the server’s public IP).
- **Port:** `5434` (same as in `docker-compose`: Postgres is mapped to 5434 on the host).
- **Database:** Same as on the server – usually `b2bapi` (or whatever `DB_NAME` is in the server’s `.env`).
- **Username:** `postgres`
- **Password:** Same as on the server – the value of `DB_PASSWORD` in the server’s `.env`.

### 2. Allow pgAdmin to reach the server (choose one)

**Option A – SSH tunnel (recommended, more secure)**  
So you don’t expose Postgres to the internet.

1. Create an SSH tunnel from your PC to the server (replace `user` and `your-server`):
   ```bash
   ssh -L 5434:127.0.0.1:5434 user@your-server
   ```
   Keep this terminal open. Now `localhost:5434` on your PC forwards to `127.0.0.1:5434` on the server (the Postgres port).
2. In pgAdmin, use:
   - **Host:** `localhost`
   - **Port:** `5434`
   - **Database:** `b2bapi`
   - **Username:** `postgres`
   - **Password:** (the server’s `DB_PASSWORD`)

**Option B – Direct connection**  
Only if the server firewall allows your IP on port 5434.

- In pgAdmin use **Host:** the server’s hostname or IP (e.g. `api.jafarshop.com`), **Port:** `5434`, then same database/user/password as above.

### 3. Add the server in pgAdmin (server DB)

1. Right‑click **Servers** → **Register** → **Server**.
2. **General** tab: **Name** e.g. `JafarShop B2B (Server)`.
3. **Connection** tab:
   - **Host:** `localhost` (if using SSH tunnel) or the server hostname/IP (if direct).
   - **Port:** `5434`
   - **Maintenance database:** `b2bapi`
   - **Username:** `postgres`
   - **Password:** (server’s `DB_PASSWORD`)
4. Optionally enable **Save password**, then **Save**.

You’re now connected to the same database the app uses on the server.

---

## B. Connect to **local** Docker database

The Postgres database runs in Docker. From your PC (where pgAdmin runs), you connect to the **host** port that forwards to the container.

### 1. Make sure the stack is running

From the repo root `b2b/`:

```bash
docker compose up -d
```

Check that the `postgres` container is up:

```bash
docker compose ps
```

You should see `b2b_postgres` with port `5434->5432`.

### 2. Connection details (from your machine)

| Field      | Value |
|-----------|--------|
| **Host**  | `localhost` (or `127.0.0.1`) |
| **Port**  | `5434` |
| **Database** | `b2bapi` (or your `DB_NAME` from `.env`) |
| **Username** | `postgres` |
| **Password** | `123123` (or your `DB_PASSWORD` from `.env`) |

The container uses port 5432 inside; Docker maps it to **5434** on the host so it doesn’t clash with a local Postgres on 5432.

### 3. Add the server in pgAdmin (local)

1. Open **pgAdmin**.
2. Right‑click **Servers** in the left tree → **Register** → **Server**.
3. **General** tab:
   - **Name:** e.g. `JafarShop B2B` (any label you like).
4. **Connection** tab:
   - **Host name/address:** `localhost`
   - **Port:** `5434`
   - **Maintenance database:** `b2bapi`
   - **Username:** `postgres`
   - **Password:** `123123` (or your actual `DB_PASSWORD`).
5. Optionally: **Connection** tab → enable **Save password** so you don’t type it every time.
6. Click **Save**.

### 4. Connect and browse

- In the left tree, expand **Servers** → **JafarShop B2B** → **Databases** → **b2bapi** → **Schemas** → **public** → **Tables**.
- You should see tables such as `supplier_orders`, `partners`, `partner_sku_mappings`, etc.

### If connection fails (local)

- **“Connection refused”**  
  - Confirm Docker is running and the postgres container is up: `docker compose ps`.  
  - Confirm port **5434** is not blocked by firewall or used by another app.

- **“Password authentication failed”**  
  - Use the same password as in your `.env`: `DB_PASSWORD`.  
  - Default in `docker-compose.yml` is `123123` if `DB_PASSWORD` is not set.

- **Database “b2bapi” does not exist**  
  - Check `.env`: `DB_NAME` (default is `b2bapi`).  
  - Create the DB in the container if you changed the name after first run.

- **Connecting from another machine**  
  - Use that machine’s IP instead of `localhost`, and ensure port 5434 is open. Prefer SSH tunnel (see section A) over exposing the DB to the internet.

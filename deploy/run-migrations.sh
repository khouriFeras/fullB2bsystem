#!/usr/bin/env bash
# Run OrderB2bAPI DB migrations once after first 'docker compose up -d'.
# Uses golang-migrate Docker image. Run from repo root (b2b/).
#
# Either:
#   - Have .env in repo root with DB_PASSWORD and DB_NAME set, then:
#     ./deploy/run-migrations.sh
#   - Or pass explicitly:
#     DB_PASSWORD=yourpass DB_NAME=b2bapi ./deploy/run-migrations.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

if [ -f .env ]; then
  set -a
  # shellcheck source=/dev/null
  source .env
  set +a
fi

DB_PASSWORD="${DB_PASSWORD:-123123}"
DB_NAME="${DB_NAME:-b2bapi}"
DB_PORT_HOST="${DB_PORT_HOST:-5434}"

echo "Running migrations (database: $DB_NAME, host port: $DB_PORT_HOST) ..."
docker run --rm \
  -v "$ROOT_DIR/OrderB2bAPI/migrations:/migrations" \
  --network host \
  migrate/migrate \
  -path /migrations \
  -database "postgres://postgres:${DB_PASSWORD}@127.0.0.1:${DB_PORT_HOST}/${DB_NAME}?sslmode=disable" \
  up

echo "Migrations done. Restart API if needed: docker compose restart orderb2bapi"

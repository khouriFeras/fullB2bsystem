#!/usr/bin/env bash
# E2E test: Create partner (optional) -> Catalog -> Cart submit -> Get order -> Delivery status.
# Run from repo root (b2b/). Requires: curl, jq (optional, for parsing).
#
# Usage:
#   Option A - Create partner then run all steps (on server, from ~/b2b):
#     COLLECTION_HANDLE=wholesale bash deploy/test-e2e.sh
#     Or: bash deploy/test-e2e.sh wholesale
#   Option B - Use existing partner API key (from anywhere):
#     API_BASE=http://95.217.6.87:8081 PARTNER_API_KEY=your-key bash deploy/test-e2e.sh
#   If ./deploy/test-e2e.sh says "No such file or directory", use: bash deploy/test-e2e.sh
#   Or fix CRLF: sed -i 's/\r$//' deploy/test-e2e.sh
#
# Env:
#   API_BASE          OrderB2bAPI base URL (default http://127.0.0.1:8081)
#   PARTNER_API_KEY   If set, skip partner creation and use this key
#   COLLECTION_HANDLE Shopify collection handle (required if creating partner)
#   SKIP_PARTNER_CREATE  If 1 and PARTNER_API_KEY is set, skip step 1
#   SHOW_RESPONSES       If 1, print full server response body after each step (so you see JSON in the terminal)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

API_BASE="${API_BASE:-http://127.0.0.1:8081}"
COLLECTION_HANDLE="${COLLECTION_HANDLE:-$1}"
PARTNER_API_KEY="${PARTNER_API_KEY:-}"
SKIP_PARTNER_CREATE="${SKIP_PARTNER_CREATE:-0}"
SHOW_RESPONSES="${SHOW_RESPONSES:-0}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
ok() { echo -e "${GREEN}[OK]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }

# --- Step 0: Health ---
echo "=== Step 0: Health check ==="
health=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/health" || true)
if [ "$health" = "200" ]; then
  ok "OrderB2bAPI health $API_BASE/health -> 200"
  [ "$SHOW_RESPONSES" = "1" ] && curl -s "$API_BASE/health" | head -c 500 && echo ""
else
  fail "OrderB2bAPI health returned $health (expected 200). Is the stack running?"
fi

# --- Step 1: Create partner (optional) ---
if [ -z "$PARTNER_API_KEY" ] && [ -n "$COLLECTION_HANDLE" ] && [ "$SKIP_PARTNER_CREATE" != "1" ]; then
  echo "=== Step 1: Create partner ==="
  API_KEY="e2e-test-api-key-$(openssl rand -hex 8 2>/dev/null || echo "e2e-fallback-$$")"
  out=$(docker compose run --rm --entrypoint ./create-partner orderb2bapi "E2E Test Partner" "$API_KEY" "$COLLECTION_HANDLE" 2>&1) || true
  if echo "$out" | grep -q "Partner created successfully"; then
    PARTNER_API_KEY="$API_KEY"
    ok "Partner created; using API key from create-partner output"
  else
    fail "create-partner failed. Output: $out"
  fi
elif [ -z "$PARTNER_API_KEY" ]; then
  fail "Set PARTNER_API_KEY=... or COLLECTION_HANDLE=... (and run from repo root with docker) to create a partner"
else
  echo "=== Step 1: Using existing PARTNER_API_KEY ==="
  ok "Skipping partner creation"
fi

KEY="$PARTNER_API_KEY"

# --- Step 2: Catalog ---
echo "=== Step 2: Catalog (GET /v1/catalog/products) ==="
catalog_resp=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $KEY" "$API_BASE/v1/catalog/products?limit=10")
catalog_http=$(echo "$catalog_resp" | tail -n1)
catalog_body=$(echo "$catalog_resp" | sed '$d')
if [ "$catalog_http" != "200" ]; then
  fail "Catalog returned HTTP $catalog_http. Body: $catalog_body"
fi
if echo "$catalog_body" | grep -q '"error"'; then
  fail "Catalog returned error: $catalog_body"
fi
# Pick first SKU for cart (jq if available, else grep/sed)
FIRST_SKU=""
if command -v jq &>/dev/null; then
  FIRST_SKU=$(echo "$catalog_body" | jq -r '.data[0].sku // empty')
fi
if [ -z "$FIRST_SKU" ]; then
  FIRST_SKU=$(echo "$catalog_body" | grep -o '"sku"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)".*/\1/')
fi
if [ -z "$FIRST_SKU" ]; then
  warn "No SKU found in catalog (empty or format unknown). Cart submit may return 204. Body (first 200 chars): ${catalog_body:0:200}"
  FIRST_SKU="REAL-SKU"
fi
ok "Catalog returned 200; first SKU for cart: $FIRST_SKU"
[ "$SHOW_RESPONSES" = "1" ] && echo "--- Catalog response ---" && echo "$catalog_body" | head -c 1500 && echo ""

# --- Step 3: Cart submit ---
echo "=== Step 3: Cart submit (POST /v1/carts/submit) ==="
ORDER_ID="E2E-ORDER-$(date +%s)"
IDEM_KEY="e2e-idem-$(date +%s)"
cart_body=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/v1/carts/submit" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEM_KEY" \
  -d "{
    \"partner_order_id\": \"$ORDER_ID\",
    \"items\": [{\"sku\": \"$FIRST_SKU\", \"title\": \"E2E Product\", \"price\": 10.00, \"quantity\": 1}],
    \"customer\": {\"first_name\": \"Test\", \"last_name\": \"User\", \"phone_number\": \"+962700000000\", \"email\": \"e2e@test.com\"},
    \"shipping\": {\"city\": \"Amman\", \"address\": \"123 St\", \"postal_code\": \"11118\", \"country\": \"JO\"},
    \"totals\": {\"subtotal\": 10.00, \"tax\": 0, \"shipping\": 0, \"total\": 10.00}
  }")
cart_http=$(echo "$cart_body" | tail -n1)
cart_json=$(echo "$cart_body" | sed '$d')
if [ "$cart_http" = "204" ]; then
  warn "Cart submit 204 (no JafarShop items matched). Order may not exist. Continuing with delivery-status check anyway."
elif [ "$cart_http" != "200" ]; then
  fail "Cart submit returned HTTP $cart_http. Body: $cart_json"
else
  ok "Cart submit 200; partner_order_id=$ORDER_ID"
fi
[ "$SHOW_RESPONSES" = "1" ] && echo "--- Cart submit response ---" && echo "$cart_json" | head -c 1000 && echo ""

# --- Step 4: Get order ---
echo "=== Step 4: Get order (GET /v1/orders/:id) ==="
order_resp=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $KEY" "$API_BASE/v1/orders/$ORDER_ID")
order_http=$(echo "$order_resp" | tail -n1)
order_json=$(echo "$order_resp" | sed '$d')
if [ "$order_http" = "200" ]; then
  ok "Get order 200"
else
  warn "Get order returned $order_http (may be 404 if cart was 204). Body: ${order_json:0:150}"
fi
[ "$SHOW_RESPONSES" = "1" ] && echo "--- Get order response ---" && echo "$order_json" | head -c 1500 && echo ""

# --- Step 5: Delivery status ---
echo "=== Step 5: Delivery status (GET /v1/orders/:id/delivery-status) ==="
delivery_resp=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $KEY" "$API_BASE/v1/orders/$ORDER_ID/delivery-status")
delivery_http=$(echo "$delivery_resp" | tail -n1)
delivery_json=$(echo "$delivery_resp" | sed '$d')
if [ "$delivery_http" = "200" ]; then
  ok "Delivery status 200 (GetDeliveryStatus/Wassel returned data or empty shipment)"
elif [ "$delivery_http" = "502" ]; then
  warn "Delivery status 502 - GetDeliveryStatus or Wassel may be unavailable or reference not found. Body: ${delivery_json:0:200}"
else
  warn "Delivery status returned HTTP $delivery_http. Body: ${delivery_json:0:150}"
fi
[ "$SHOW_RESPONSES" = "1" ] && echo "--- Delivery status response ---" && echo "$delivery_json" | head -c 1500 && echo ""

echo ""
echo "=== E2E test finished ==="
echo "  Partner API key (for manual retries): $KEY"
echo "  Partner order id: $ORDER_ID"
echo "  Catalog: first SKU used: $FIRST_SKU"

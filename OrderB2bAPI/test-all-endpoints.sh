#!/bin/bash
# Test all OrderB2bAPI endpoints
# Usage: ./test-all-endpoints.sh
#        PARTNER_API_KEY=partner123 ./test-all-endpoints.sh
#        ORDER_B2B_API_URL=https://api.jafarshop.com PARTNER_API_KEY=partner123 ./test-all-endpoints.sh

BASE_URL="${ORDER_B2B_API_URL:-http://localhost:8081}"
API_KEY="${PARTNER_API_KEY:-}"
if [ -z "$API_KEY" ] && [ -f "../.env" ]; then
  API_KEY=$(grep -E "PARTNER_API_KEYS=" ../.env 2>/dev/null | head -1 | cut -d= -f2 | cut -d: -f2)
fi

if [ -z "$API_KEY" ]; then
  echo "ERROR: PARTNER_API_KEY not set."
  echo "  Set: PARTNER_API_KEY=partner123 ./test-all-endpoints.sh"
  echo "  Or create a partner: docker compose run --rm orderb2bapi ./create-partner 'Test Partner' 'your-api-key'"
  exit 1
fi

SKU="TEST-SKU"
PASS=0
FAIL=0

test_ep() {
  local name="$1"
  local url="$2"
  local method="${3:-GET}"
  local expected="$4"
  local use_auth="${5:-}"
  local body="${6:-}"
  echo ""
  echo "--- $name ---"
  echo "  $method $url"
  if [ "$use_auth" = "auth" ]; then
    if [ "$method" = "POST" ] && [ -n "$body" ]; then
      idem="${7:-}"
      [ -z "$idem" ] && idem="test-idem-$(date +%s)"
      code=$(curl -s -o /tmp/out.json -w "%{http_code}" -X POST \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -H "Idempotency-Key: $idem" \
        -d "$body" "$url")
    else
      code=$(curl -s -o /tmp/out.json -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$url")
    fi
  else
    code=$(curl -s -o /tmp/out.json -w "%{http_code}" "$url")
  fi
  if [ "$code" = "$expected" ]; then
    echo "  Status: $code OK (expected)"
    ((PASS++))
  else
    echo "  Status: $code (expected $expected)"
    ((FAIL++))
  fi
  head -c 150 /tmp/out.json 2>/dev/null | tr '\n' ' '
  echo "..."
}

echo "OrderB2bAPI endpoint tests - Base: $BASE_URL"
echo "API key: ${API_KEY:0:12}..."

test_ep "1. GET /health" "$BASE_URL/health" "GET" 200
test_ep "2. GET /v1/catalog/products?limit=5" "$BASE_URL/v1/catalog/products?limit=5" "GET" 200 auth

# Get first SKU from catalog response for cart submit
if [ -f /tmp/out.json ] && command -v jq &>/dev/null; then
  SKU=$(jq -r '.data[0].variants.nodes[0].sku // .data[0].sku // .data[0].variants[0].sku // "TEST-SKU"' /tmp/out.json 2>/dev/null)
fi
[ -z "$SKU" ] || [ "$SKU" = "null" ] && SKU="TEST-SKU"

# 3. Cart submit
PARTNER_ORDER_ID="test-order-$(date +%Y%m%d-%H%M%S)"
CART_JSON="{\"partner_order_id\":\"$PARTNER_ORDER_ID\",\"items\":[{\"sku\":\"$SKU\",\"title\":\"Test Product\",\"price\":10,\"quantity\":1}],\"customer\":{\"first_name\":\"Test\",\"last_name\":\"User\",\"email\":\"test@example.com\",\"phone_number\":\"+962700000000\"},\"shipping\":{\"city\":\"Amman\",\"area\":\"Downtown\",\"address\":\"123 Test St\",\"postal_code\":\"11181\",\"country\":\"Jordan\"},\"totals\":{\"subtotal\":10,\"tax\":0,\"shipping\":0,\"total\":10}}"
test_ep "3. POST /v1/carts/submit" "$BASE_URL/v1/carts/submit" "POST" 200 auth "$CART_JSON"

test_ep "4. GET /v1/admin/orders?limit=5" "$BASE_URL/v1/admin/orders?limit=5" "GET" 200 auth
test_ep "5. GET /v1/admin/orders?status=PENDING_CONFIRMATION&limit=3" "$BASE_URL/v1/admin/orders?status=PENDING_CONFIRMATION&limit=3" "GET" 200 auth
test_ep "6. GET /v1/catalog/products?limit=2" "$BASE_URL/v1/catalog/products?limit=2" "GET" 200 auth
test_ep "7. GET /v1/catalog/products (no auth)" "$BASE_URL/v1/catalog/products?limit=1" "GET" 401

echo ""
echo "--- Summary: $PASS passed, $FAIL failed ---"

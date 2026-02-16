#!/bin/bash
# Test all ProductB2B API endpoints
# Usage: ./test-all-endpoints.sh
#        PARTNER_API_KEY=partner123 ./test-all-endpoints.sh
#
# Set API key from server .env (PARTNER_API_KEYS or PRODUCT_B2B_SERVICE_API_KEY)

BASE_URL="${PRODUCT_B2B_URL:-https://products.jafarshop.com}"
API_KEY="${PARTNER_API_KEY:-}"
if [ -z "$API_KEY" ] && [ -f "../.env" ]; then
  API_KEY=$(grep -E "PRODUCT_B2B_SERVICE_API_KEY=|PARTNER_API_KEYS=" ../.env 2>/dev/null | head -1 | cut -d= -f2 | cut -d: -f2)
fi
API_KEY="${API_KEY:-partner123}"
SKU="MK4820b"
PASS=0
FAIL=0

test_ep() {
  local name="$1"
  local url="$2"
  local expected="$3"
  local use_auth="${4:-}"
  echo ""
  echo "--- $name ---"
  echo "  GET $url"
  if [ "$use_auth" = "auth" ]; then
    code=$(curl -s -o /tmp/out.json -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$url")
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
  head -c 120 /tmp/out.json 2>/dev/null | tr '\n' ' '
  echo "..."
}

echo "ProductB2B endpoint tests - Base: $BASE_URL"
echo "API key: ${API_KEY:0:12}..."

test_ep "1. GET /health" "$BASE_URL/health" 200
test_ep "2. GET /v1/catalog/products?limit=25&lang=en" "$BASE_URL/v1/catalog/products?limit=25&lang=en" 200 auth
test_ep "3. GET /v1/catalog/products?limit=25&lang=ar" "$BASE_URL/v1/catalog/products?limit=25&lang=ar" 200 auth
test_ep "4. GET /v1/catalog/products?sku=$SKU" "$BASE_URL/v1/catalog/products?sku=$SKU" 403 auth
test_ep "5. GET /menus" "$BASE_URL/menus" 200
test_ep "6. GET /menu-path-by-sku?sku=$SKU" "$BASE_URL/menu-path-by-sku?sku=$SKU" 200
test_ep "7. GET /debug/sku-lookup?sku=$SKU" "$BASE_URL/debug/sku-lookup?sku=$SKU" 200

# Get product GID from sku-lookup for translations
PRODUCT_GID=$(curl -s "$BASE_URL/debug/sku-lookup?sku=$SKU" | grep -o 'gid://shopify/Product/[0-9]*' | head -1)
if [ -n "$PRODUCT_GID" ]; then
  ENCODED_GID=$(echo -n "$PRODUCT_GID" | python3 -c "import sys,urllib.parse; print(urllib.parse.quote(sys.stdin.read().strip()))" 2>/dev/null || echo "$PRODUCT_GID")
  test_ep "8. GET /debug/partner-products" "$BASE_URL/debug/partner-products" 200
  test_ep "9. GET /debug/menu" "$BASE_URL/debug/menu" 200
  test_ep "10. GET /debug/translations" "$BASE_URL/debug/translations?product_id=$ENCODED_GID&locale=en" 200
else
  test_ep "8. GET /debug/partner-products" "$BASE_URL/debug/partner-products" 200
  test_ep "9. GET /debug/menu" "$BASE_URL/debug/menu" 200
  test_ep "10. GET /debug/translations" "$BASE_URL/debug/translations?product_id=gid%3A%2F%2Fshopify%2FProduct%2F9049440125140&locale=en" 200
fi

echo ""
echo "--- Summary: $PASS passed, $FAIL failed ---"

# ProductB2B API – Endpoint Tests

Copy and paste these commands into your terminal to test each endpoint. Use `https://products.jafarshop.com` for production, or `http://localhost:3000` for local.

Replace `YOUR_API_KEY` with a valid key from your server `.env` (e.g. `partner123` or `pycQSpnhaE`).

**Windows PowerShell:** Use `curl.exe` instead of `curl` (PowerShell aliases `curl` to `Invoke-WebRequest`). Or use the `Invoke-RestMethod` examples below.

---

## 1. Health check

Checks if the ProductB2B service is running. Returns `ok` if alive. No auth.

**Bash / Git Bash / WSL:**

```bash
curl -s "https://products.jafarshop.com/health"
```

**PowerShell:**

```powershell
Invoke-RestMethod -Uri "https://products.jafarshop.com/health"
```

**PowerShell (with curl.exe):**

```powershell
curl.exe -s "https://products.jafarshop.com/health"
```

Expected: `ok` (HTTP 200)

---

## 2. Catalog products (English)

First 25 products from the Partner Catalog in English. Requires Bearer token.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" -H "Authorization: Bearer partner123" "https://products.jafarshop.com/v1/catalog/products?limit=25&lang=en"
```

Expected: JSON with `data` array (HTTP 200)

---

## 3. Catalog products (Arabic)

Same as above with Arabic translations.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" -H "Authorization: Bearer partner123" "https://products.jafarshop.com/v1/catalog/products?limit=25&lang=ar"
```

Expected: JSON with translated fields (HTTP 200)

---

## 4. Single product by SKU

Fetches one product by SKU. Use `403` if product exists but is not in Partner Catalog.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" -H "Authorization: Bearer partner123" "https://products.jafarshop.com/v1/catalog/products?sku=MK4820b"
```

Expected: 403 (product not in catalog) or 200 (product in catalog)

---

## 5. Menus

All navigation menus from Shopify (nested items). No auth.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" "https://products.jafarshop.com/menus"
```

Expected: JSON with `menus` array (HTTP 200)

---

## 6. Menu path by SKU

Product and its menu hierarchy (breadcrumbs) for a given SKU. **Requires auth** (Bearer token: partner or service API key).

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" -H "Authorization: Bearer YOUR_API_KEY" "https://products.jafarshop.com/menu-path-by-sku?sku=MK4820b"
```

Expected: JSON with `productId`, `productName`, `menuPath` (HTTP 200). Without valid token: HTTP 401.

---

## 7. Debug SKU lookup

Lookup product GID by SKU. No auth.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" "https://products.jafarshop.com/debug/sku-lookup?sku=2899"
```

Expected: `{"found":true,"productId":"gid://shopify/Product/...","sku":"MK4820b"}` (HTTP 200)

---

## 8. Debug partner products

Raw Partner Catalog data from Shopify GraphQL. No auth.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" "https://products.jafarshop.com/debug/partner-products"
```

Expected: JSON with collection and products (HTTP 200)

---

## 9. Debug menu

Raw main menu structure. No auth.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" "https://products.jafarshop.com/debug/menu"
```

Expected: JSON with menu structure (HTTP 200)

---

## 10. Debug translations

Translated fields for a product and locale. Use `product_id` from sku-lookup. No auth.

```bash
curl.exe -s -w "\nHTTP %{http_code}\n" "https://products.jafarshop.com/debug/translations?product_id=gid://shopify/Product/9049440125140&locale=en"
```

Expected: JSON with `translations` (HTTP 200)

---

## One-liner: run all tests (status codes)

**PowerShell:** (use `curl.exe` so it doesn't alias to Invoke-WebRequest)

```powershell
$API_KEY="YOUR_API_KEY"; $BASE="https://products.jafarshop.com"
Write-Host "1. Health: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" $BASE/health
Write-Host "`n2. Catalog en: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE/v1/catalog/products?limit=25&lang=en"
Write-Host "`n3. Catalog ar: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE/v1/catalog/products?limit=25&lang=ar"
Write-Host "`n4. SKU: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE/v1/catalog/products?sku=MK4820b"
Write-Host "`n5. Menus: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/menus"
Write-Host "`n6. Menu path: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/menu-path-by-sku?sku=MK4820b"
Write-Host "`n7. SKU lookup: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/debug/sku-lookup?sku=MK4820b"
Write-Host "`n8. Partner products: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/debug/partner-products"
Write-Host "`n9. Debug menu: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/debug/menu"
Write-Host "`n10. Translations: " -NoNewline; curl.exe -s -o NUL -w "%{http_code}" "$BASE/debug/translations?product_id=gid://shopify/Product/9049440125140&locale=en"
```

**Bash / Git Bash:**

```bash
API_KEY="YOUR_API_KEY"
BASE="https://products.jafarshop.com"
echo "1. Health: $(curl.exe -s -o /dev/null -w "%{http_code}" $BASE/health)"
echo "2. Catalog en: $(curl.exe -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE/v1/catalog/products?limit=25&lang=en")"
echo "3. Catalog ar: $(curl.exe -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE/v1/catalog/products?limit=25&lang=ar")"
echo "4. SKU: $(curl.exe -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE/v1/catalog/products?sku=MK4820b")"
echo "5. Menus: $(curl.exe -s -o /dev/null -w "%{http_code}" $BASE/menus)"
echo "6. Menu path: $(curl.exe -s -o /dev/null -w "%{http_code}" $BASE/menu-path-by-sku?sku=MK4820b)"
echo "7. SKU lookup: $(curl.exe -s -o /dev/null -w "%{http_code}" $BASE/debug/sku-lookup?sku=MK4820b)"
echo "8. Partner products: $(curl.exe -s -o /dev/null -w "%{http_code}" $BASE/debug/partner-products)"
echo "9. Debug menu: $(curl.exe -s -o /dev/null -w "%{http_code}" $BASE/debug/menu)"
echo "10. Translations: $(curl.exe -s -o /dev/null -w "%{http_code}" "$BASE/debug/translations?product_id=gid://shopify/Product/9049440125140&locale=en")"
```

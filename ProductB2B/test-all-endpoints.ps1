# Test all ProductB2B API endpoints
# Usage: .\test-all-endpoints.ps1
#
# Local:  $env:PRODUCT_B2B_URL="http://localhost:3000"; .\test-all-endpoints.ps1
# Prod:   .\test-all-endpoints.ps1  (defaults to https://products.jafarshop.com)
#
# API key: from .env or set $env:PARTNER_API_KEY="your-key"
# Optional: $env:ADMIN_SETUP_KEY="..." to test POST /admin/setup/webhooks (test 15)
# Save responses to JSON: $env:SAVE_RESPONSES_JSON=".\productb2b-responses.json"; .\test-all-endpoints.ps1

$ErrorActionPreference = "Stop"
$baseUrl = if ($env:PRODUCT_B2B_URL) { $env:PRODUCT_B2B_URL.TrimEnd('/') } else { "https://products.jafarshop.com" }
$script:saveResponses = @()

# API key: env override > .env > fallback
$apiKey = $env:PARTNER_API_KEY
if (-not $apiKey) {
    if (Test-Path "$PSScriptRoot\..\.env") {
        Get-Content "$PSScriptRoot\..\.env" | ForEach-Object {
            if ($_ -match "PRODUCT_B2B_SERVICE_API_KEY=(.+)") { $apiKey = $matches[1].Trim() }
            if (-not $apiKey -and $_ -match "PARTNER_API_KEYS=([^#]+)") {
                $keys = $matches[1].Trim()
                if ($keys -match "([^:,]+):([^,]+)") { $apiKey = $matches[2].Trim() }
            }
        }
    }
}
if (-not $apiKey) { $apiKey = "dev-key-123" }

$headers = @{ "Authorization" = "Bearer $apiKey" }
$sku = "MK4820b"
$script:passCount = 0
$script:failCount = 0

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Url,
        [hashtable]$Hdrs = @{},
        [string]$Method = "GET",
        [object]$Body = $null,
        [int]$ExpectedStatus = 200,
        [string]$Explanation = ""
    )
    Write-Host "`n========================================" -ForegroundColor DarkCyan
    Write-Host "  $Name" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor DarkCyan
    if ($Explanation) {
        Write-Host "  What it tests: $Explanation" -ForegroundColor White
    }
    Write-Host "  URL: $Method $Url" -ForegroundColor Gray
    Write-Host "  Expected status: $ExpectedStatus" -ForegroundColor Gray
    Write-Host ""

    try {
        $params = @{ Uri = $Url; Method = $Method; UseBasicParsing = $true; TimeoutSec = 30 }
        if ($Hdrs.Count -gt 0) { $params.Headers = $Hdrs }
        if ($Body -ne $null) { $params.Body = $Body }
        $r = Invoke-WebRequest @params -ErrorAction Stop
        $statusCode = $r.StatusCode
        $content = if ($r.Content.Length -gt 200) { $r.Content.Substring(0, 200) + "..." } else { $r.Content }
        # Prefer a short summary for /menus (hierarchical JSON doesn't truncate well)
        if ($Url -match '/menus' -and $r.Content -and $statusCode -eq 200) {
            try {
                $j = $r.Content | ConvertFrom-Json
                if ($j.menus -and $j.menus.Count -gt 0) {
                    $m = $j.menus[0]
                    $topLevel = if ($m.items) { $m.items.Count } else { 0 }
                    $content = "count=$($j.count), menus[0]: handle=$($m.handle), title=$($m.title), items: $topLevel top-level (hierarchical: title + children)"
                }
            } catch {}
        }

        $ok = ($statusCode -eq $ExpectedStatus)
        if ($ok) {
            $script:passCount++
            Write-Host "  Result: PASS - Status $statusCode (expected)" -ForegroundColor Green
        } else {
            $script:failCount++
            Write-Host "  Result: FAIL - Status $statusCode (expected $ExpectedStatus)" -ForegroundColor Red
        }
        Write-Host "  Response preview: $content" -ForegroundColor DarkGray
        if ($env:SAVE_RESPONSES_JSON -and $r.Content) {
            $responseBody = try { $r.Content | ConvertFrom-Json } catch { $r.Content }
            $script:saveResponses += @{ name = $Name; url = $Url; method = $Method; statusCode = $statusCode; pass = $ok; response = $responseBody }
        }
        return @{ Response = $r; StatusCode = $statusCode }
    } catch {
        $statusCode = 0
        $body = ""
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $reader.BaseStream.Position = 0
                $body = $reader.ReadToEnd()
                $reader.Close()
            } catch {}
        }
        $ok = ($statusCode -eq $ExpectedStatus)
        if ($ok) {
            $script:passCount++
            Write-Host "  Result: PASS - Status $statusCode (expected, e.g. 403 Forbidden)" -ForegroundColor Green
        } else {
            $script:failCount++
            Write-Host "  Result: FAIL - Status $statusCode (expected $ExpectedStatus)" -ForegroundColor Red
        }
        if ($body) { Write-Host "  Response: $($body.Substring(0, [Math]::Min(150, $body.Length)))..." -ForegroundColor DarkGray }
        if ($env:SAVE_RESPONSES_JSON) {
            $responseBody = try { $body | ConvertFrom-Json } catch { $body }
            $script:saveResponses += @{ name = $Name; url = $Url; method = $Method; statusCode = $statusCode; pass = $ok; response = $responseBody }
        }
        return @{ Response = $null; StatusCode = $statusCode }
    }
}

Write-Host "`n" -NoNewline
Write-Host " PRODUCTB2B ENDPOINT TEST SUITE " -ForegroundColor Black -BackgroundColor Yellow
Write-Host " Base URL: $baseUrl" -ForegroundColor Yellow
Write-Host " API Key: $($apiKey.Substring(0,[Math]::Min(12,$apiKey.Length)))..." -ForegroundColor DarkGray

# ---------------------------------------------------------------------------
# 1. Health check
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "1. GET /health" `
    -Url "$baseUrl/health" `
    -ExpectedStatus 200 `
    -Explanation "Checks if the ProductB2B service is running. Returns 'ok' if alive. No auth required."

# ---------------------------------------------------------------------------
# 2. Catalog products (English)
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "2. GET /v1/catalog/products?limit=25&lang=en" `
    -Url "$baseUrl/v1/catalog/products?limit=25&lang=en" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Returns the first 25 products from the Partner Catalog in English. Requires Bearer token (partner or service API key). Partners see only products in their assigned collection."

# ---------------------------------------------------------------------------
# 3. Catalog products (Arabic)
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "3. GET /v1/catalog/products?limit=25&lang=ar" `
    -Url "$baseUrl/v1/catalog/products?limit=25&lang=ar" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Same as #2 but with Arabic translations (title, description, etc.). Used for Arabic storefronts."

# ---------------------------------------------------------------------------
# 4. Single product by SKU (not in Partner Catalog)
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "4. GET /v1/catalog/products?sku=$sku" `
    -Url "$baseUrl/v1/catalog/products?sku=$sku" `
    -Hdrs $headers `
    -ExpectedStatus 403 `
    -Explanation "Fetches a single product by SKU. Returns 403 if the product exists but is NOT in the Partner Catalog collection (access denied). Returns 200 if the product is in the catalog. Tests that partners can only access catalog products."

# ---------------------------------------------------------------------------
# 5. Menus – main-menu, titles only
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "5. GET /menus" `
    -Url "$baseUrl/menus" `
    -ExpectedStatus 200 `
    -Explanation "Returns main-menu only, hierarchical items (title + children). No auth."

# ---------------------------------------------------------------------------
# 6. Menu path by SKU
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "6. GET /menu-path-by-sku?sku=$sku" `
    -Url "$baseUrl/menu-path-by-sku?sku=$sku" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Given a product SKU, returns the product and its location in the menu hierarchy. Requires Bearer token (partner or service API key)."

# ---------------------------------------------------------------------------
# 7. Debug SKU lookup
# ---------------------------------------------------------------------------
$skuResult = Test-Endpoint `
    -Name "7. GET /debug/sku-lookup?sku=$sku" `
    -Url "$baseUrl/debug/sku-lookup?sku=$sku" `
    -ExpectedStatus 200 `
    -Explanation "Debug endpoint: looks up a product by SKU and returns its Shopify GID. No auth. Used for testing and to get product_id for other calls (e.g. translations)."
$productGid = $null
if ($skuResult.Response -and $skuResult.Response.Content) {
    try {
        $j = $skuResult.Response.Content | ConvertFrom-Json
        $productGid = $j.productId
        if ($productGid) { Write-Host "  Extracted productId: $productGid" -ForegroundColor Gray }
    } catch {}
}

# ---------------------------------------------------------------------------
# 8. Debug partner products
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "8. GET /debug/partner-products" `
    -Url "$baseUrl/debug/partner-products" `
    -ExpectedStatus 200 `
    -Explanation "Debug: raw Partner Catalog collection and products from Shopify GraphQL. Shows the default collection used when no partner-specific collection is set. No auth."

# ---------------------------------------------------------------------------
# 9. Debug menu
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "9. GET /debug/menu" `
    -Url "$baseUrl/debug/menu" `
    -ExpectedStatus 200 `
    -Explanation "Debug: raw main menu structure from Shopify. Same data as /menus but in different format. No auth."

# ---------------------------------------------------------------------------
# 10. Debug translations
# ---------------------------------------------------------------------------
if ($productGid) {
    Test-Endpoint `
        -Name "10. GET /debug/translations?product_id=GID&locale=en" `
        -Url "$baseUrl/debug/translations?product_id=$([uri]::EscapeDataString($productGid))&locale=en" `
        -ExpectedStatus 200 `
        -Explanation "Debug: returns translated fields (title, body_html, vendor, product_type) for a product in a given locale (en, ar). Requires product GID from sku-lookup. No auth."
} else {
    Test-Endpoint `
        -Name "10. GET /debug/translations?product_id=...&locale=en" `
        -Url "$baseUrl/debug/translations?product_id=gid://shopify/Product/9049440125140&locale=en" `
        -ExpectedStatus 200 `
        -Explanation "Debug: returns translated product fields. Using fallback product ID since sku-lookup did not return one."
}

# ---------------------------------------------------------------------------
# 11. Root (server info)
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "11. GET / (root)" `
    -Url "$baseUrl/" `
    -ExpectedStatus 200 `
    -Explanation "Root path returns server info and list of endpoints. No auth."

# ---------------------------------------------------------------------------
# 12. Debug list collections
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "12. GET /debug/list-collections" `
    -Url "$baseUrl/debug/list-collections" `
    -ExpectedStatus 200 `
    -Explanation "Debug: lists all Shopify collections (GraphQL). Uses app token. No auth."

# ---------------------------------------------------------------------------
# 13. Debug access scopes
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "13. GET /debug/access-scopes" `
    -Url "$baseUrl/debug/access-scopes" `
    -ExpectedStatus 200 `
    -Explanation "Debug: returns OAuth access scopes for the app. Uses app token. May return 500 if Shopify unreachable."

# ---------------------------------------------------------------------------
# 14. Debug inventory status
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "14. GET /debug/inventory-status" `
    -Url "$baseUrl/debug/inventory-status" `
    -ExpectedStatus 200 `
    -Explanation "Debug: inventory status for Partner Catalog products. No auth."

# ---------------------------------------------------------------------------
# 15. POST /admin/setup/webhooks (optional: needs X-Setup-Key)
# ---------------------------------------------------------------------------
$setupKey = $env:ADMIN_SETUP_KEY
if ($setupKey) {
    $setupHeaders = @{ "Authorization" = "Bearer $apiKey"; "X-Setup-Key" = $setupKey; "Content-Type" = "application/json" }
    Test-Endpoint `
        -Name "15. POST /admin/setup/webhooks" `
        -Url "$baseUrl/admin/setup/webhooks" `
        -Hdrs $setupHeaders `
        -Method "POST" `
        -Body "{}" `
        -ExpectedStatus 200 `
        -Explanation "Registers Shopify webhooks. Requires X-Setup-Key from ADMIN_SETUP_KEY. Returns webhook topic status."
} else {
    # Without key: expect 401
    Test-Endpoint `
        -Name "15. POST /admin/setup/webhooks (no key)" `
        -Url "$baseUrl/admin/setup/webhooks" `
        -Hdrs @{ "Content-Type" = "application/json" } `
        -Method "POST" `
        -Body "{}" `
        -ExpectedStatus 401 `
        -Explanation "Without X-Setup-Key returns 401. Set env ADMIN_SETUP_KEY to test full registration."
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
Write-Host "`n========================================" -ForegroundColor DarkCyan
Write-Host " SUMMARY " -ForegroundColor Black -BackgroundColor $(if ($script:failCount -eq 0) { "Green" } else { "Yellow" })
Write-Host "========================================" -ForegroundColor DarkCyan
Write-Host "  Passed: $script:passCount" -ForegroundColor Green
Write-Host "  Failed: $script:failCount" -ForegroundColor $(if ($script:failCount -gt 0) { "Red" } else { "Green" })
if ($script:failCount -gt 0) {
    Write-Host "`n  Tip: Catalog endpoints (2,3,4) need a valid API key." -ForegroundColor Yellow
    Write-Host "  Set: `$env:PARTNER_API_KEY='partner123'" -ForegroundColor Yellow
}

if ($env:SAVE_RESPONSES_JSON -and $script:saveResponses.Count -gt 0) {
    $outPath = $env:SAVE_RESPONSES_JSON
    $script:saveResponses | ConvertTo-Json -Depth 25 | Set-Content -Path $outPath -Encoding UTF8
    Write-Host "`n  Responses saved to: $outPath" -ForegroundColor Green
}
Write-Host ""

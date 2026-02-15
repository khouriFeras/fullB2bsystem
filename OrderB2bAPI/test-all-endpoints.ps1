# Test all OrderB2bAPI endpoints and functionality
# Usage: .\test-all-endpoints.ps1
#
# Set partner API key (from create-partner):
#   $env:PARTNER_API_KEY="your-key"; .\test-all-endpoints.ps1
#
# Production/remote: Set base URL and key
#   $env:ORDER_B2B_API_URL="https://api.jafarshop.com"; $env:PARTNER_API_KEY="..."; .\test-all-endpoints.ps1

$ErrorActionPreference = "Stop"
$baseUrl = if ($env:ORDER_B2B_API_URL) { $env:ORDER_B2B_API_URL.TrimEnd('/') } else { "http://localhost:8081" }

# Partner API key: required for /v1/* endpoints (create with: go run cmd/create-partner/main.go "Name" "key")
$apiKey = $env:PARTNER_API_KEY
if (-not $apiKey) {
    Write-Host "ERROR: PARTNER_API_KEY not set." -ForegroundColor Red
    Write-Host "  Create a partner: go run cmd/create-partner/main.go `"Test Partner`" `"your-api-key`"" -ForegroundColor Yellow
    Write-Host "  Then run: `$env:PARTNER_API_KEY='your-api-key'; .\test-all-endpoints.ps1" -ForegroundColor Yellow
    exit 1
}

$headers = @{ "Authorization" = "Bearer $apiKey" }
$script:passCount = 0
$script:failCount = 0
$script:createdOrderId = $null
$script:createdPartnerOrderId = $null

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Url,
        [string]$Method = "GET",
        [hashtable]$Hdrs = @{},
        [hashtable]$Body = $null,
        [int]$ExpectedStatus = 200,
        [int[]]$ExpectedStatuses = $null,  # optional: accept multiple (e.g. 200,503)
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
        if ($Body -and ($Method -eq "POST" -or $Method -eq "PUT")) {
            $params.ContentType = "application/json"
            $params.Body = ($Body | ConvertTo-Json -Depth 10 -Compress)
        }
        $r = Invoke-WebRequest @params -ErrorAction Stop
        $statusCode = $r.StatusCode
        $content = if ($r.Content.Length -gt 250) { $r.Content.Substring(0, 250) + "..." } else { $r.Content }

        $expectedList = if ($ExpectedStatuses) { $ExpectedStatuses } else { @($ExpectedStatus) }
        $ok = $expectedList -contains $statusCode
        if ($ok) {
            $script:passCount++
            Write-Host "  Result: PASS - Status $statusCode (expected)" -ForegroundColor Green
        } else {
            $script:failCount++
            $expStr = (($expectedList -join ","))
            Write-Host "  Result: FAIL - Status $statusCode (expected $expStr)" -ForegroundColor Red
        }
        Write-Host "  Response preview: $content" -ForegroundColor DarkGray
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
        $expectedList = if ($ExpectedStatuses) { $ExpectedStatuses } else { @($ExpectedStatus) }
        $ok = $expectedList -contains $statusCode
        if ($ok) {
            $script:passCount++
            Write-Host "  Result: PASS - Status $statusCode (expected)" -ForegroundColor Green
        } else {
            $script:failCount++
            $expStr = ($expectedList -join ",")
            Write-Host "  Result: FAIL - Status $statusCode (expected $expStr)" -ForegroundColor Red
        }
        if ($body) { Write-Host "  Response: $($body.Substring(0, [Math]::Min(200, $body.Length)))..." -ForegroundColor DarkGray }
        return @{ Response = $null; StatusCode = $statusCode }
    }
}

Write-Host "`n" -NoNewline
Write-Host " ORDERB2BAPI ENDPOINT TEST SUITE " -ForegroundColor Black -BackgroundColor Cyan
Write-Host " Base URL: $baseUrl" -ForegroundColor Cyan
Write-Host " API Key: $($apiKey.Substring(0,[Math]::Min(12,$apiKey.Length)))..." -ForegroundColor DarkGray

# ---------------------------------------------------------------------------
# 1. Health check
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "1. GET /health" `
    -Url "$baseUrl/health" `
    -ExpectedStatus 200 `
    -Explanation "Checks if the OrderB2bAPI service is running. Returns {status: ok}. No auth required."

# ---------------------------------------------------------------------------
# 2. Catalog products (partner auth)
# ---------------------------------------------------------------------------
$catalogResult = Test-Endpoint `
    -Name "2. GET /v1/catalog/products?limit=5" `
    -Url "$baseUrl/v1/catalog/products?limit=5" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Returns the partner's product catalog (paginated). Requires partner Bearer token. Data comes from ProductB2B or fallback to partner_sku_mappings."

$firstSku = "TEST-SKU"
if ($catalogResult.Response -and $catalogResult.Response.Content) {
    try {
        $catalogJson = $catalogResult.Response.Content | ConvertFrom-Json
        if ($catalogJson.data -and $catalogJson.data.Count -gt 0) {
            $prod = $catalogJson.data[0]
            $firstSku = if ($prod.variants -and $prod.variants.nodes -and $prod.variants.nodes.Count -gt 0) {
                $prod.variants.nodes[0].sku
            } elseif ($prod.sku) { $prod.sku } else { "TEST-SKU" }
            Write-Host "  Extracted first SKU: $firstSku" -ForegroundColor Gray
        }
    } catch {}
}

# ---------------------------------------------------------------------------
# 3. Cart submit (create order)
# ---------------------------------------------------------------------------
$partnerOrderId = "test-order-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
$idemKey = "test-idem-" + [guid]::NewGuid().ToString("N").Substring(0, 12)
$cartPayload = @{
    partner_order_id = $partnerOrderId
    items = @(
        @{
            sku = $firstSku
            title = "Test Product"
            price = 10.00
            quantity = 1
        }
    )
    customer = @{
        first_name = "Test"
        last_name = "User"
        email = "test@example.com"
        phone_number = "+962700000000"
    }
    shipping = @{
        city = "Amman"
        area = "Downtown"
        address = "123 Test St"
        postal_code = "11181"
        country = "Jordan"
    }
    totals = @{
        subtotal = 10.00
        tax = 0
        shipping = 0
        total = 10.00
    }
}

$submitHdrs = @{ "Authorization" = "Bearer $apiKey"; "Idempotency-Key" = $idemKey }
$cartResult = Test-Endpoint `
    -Name "3. POST /v1/carts/submit" `
    -Url "$baseUrl/v1/carts/submit" `
    -Method POST `
    -Hdrs $submitHdrs `
    -Body $cartPayload `
    -ExpectedStatus 200 `
    -Explanation "Submits a cart and creates an order. Requires Idempotency-Key. SKU must exist in partner catalog. May return 422 if SKU not in catalog."

if ($cartResult.Response -and $cartResult.Response.Content -and $cartResult.StatusCode -eq 200) {
    try {
        $cartResp = $cartResult.Response.Content | ConvertFrom-Json
        $script:createdOrderId = $cartResp.supplier_order_id
        $script:createdPartnerOrderId = $cartResp.partner_order_id
        if ($script:createdOrderId) {
            Write-Host "  Created order ID: $script:createdOrderId" -ForegroundColor Gray
        }
    } catch {}
}

# ---------------------------------------------------------------------------
# 4. List orders (admin)
# ---------------------------------------------------------------------------
$listResult = Test-Endpoint `
    -Name "4. GET /v1/admin/orders?limit=5" `
    -Url "$baseUrl/v1/admin/orders?limit=5" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Lists orders for the authenticated partner with pagination. Optional status filter."

$orderIdForGet = $script:createdOrderId
if (-not $orderIdForGet -and $listResult.Response -and $listResult.Response.Content) {
    try {
        $listJson = $listResult.Response.Content | ConvertFrom-Json
        if ($listJson.orders -and $listJson.orders.Count -gt 0) {
            $orderIdForGet = $listJson.orders[0].id
            Write-Host "  Using first order from list: $orderIdForGet" -ForegroundColor Gray
        }
    } catch {}
}
if (-not $orderIdForGet) { $orderIdForGet = "00000000-0000-0000-0000-000000000000" }

# ---------------------------------------------------------------------------
# 5. Get order by ID
# ---------------------------------------------------------------------------
$getOrderUrl = "$baseUrl/v1/orders/$orderIdForGet"
$getExpected = if ($orderIdForGet -eq "00000000-0000-0000-0000-000000000000") { 404 } else { 200 }
Test-Endpoint `
    -Name "5. GET /v1/orders/:id" `
    -Url $getOrderUrl `
    -Hdrs $headers `
    -ExpectedStatus $getExpected `
    -Explanation "Returns a single order by supplier_order_id (UUID) or partner_order_id. 404 if not found, 403 if belongs to another partner."

# ---------------------------------------------------------------------------
# 6. Delivery status
# ---------------------------------------------------------------------------
$deliveryId = $script:createdOrderId
if (-not $deliveryId) { $deliveryId = $script:createdPartnerOrderId }
if (-not $deliveryId) { $deliveryId = $orderIdForGet }
$deliveryUrl = "$baseUrl/v1/orders/$deliveryId/delivery-status"
Test-Endpoint `
    -Name "6. GET /v1/orders/:id/delivery-status" `
    -Url $deliveryUrl `
    -Hdrs $headers `
    -ExpectedStatuses @(200, 503) `
    -Explanation "Returns delivery/shipment status for an order. 503 if GetDeliveryStatus service is not configured."

# ---------------------------------------------------------------------------
# 7. List orders with status filter
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "7. GET /v1/admin/orders?status=PENDING_CONFIRMATION&limit=3" `
    -Url "$baseUrl/v1/admin/orders?status=PENDING_CONFIRMATION&limit=3" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Lists orders filtered by status (PENDING_CONFIRMATION, CONFIRMED, REJECTED, SHIPPED)."

# ---------------------------------------------------------------------------
# 8. Catalog pagination (cursor)
# ---------------------------------------------------------------------------
$cursorResult = Test-Endpoint `
    -Name "8. GET /v1/catalog/products?limit=2&cursor=" `
    -Url "$baseUrl/v1/catalog/products?limit=2" `
    -Hdrs $headers `
    -ExpectedStatus 200 `
    -Explanation "Catalog with pagination. Use nextCursor from response for subsequent pages."

# ---------------------------------------------------------------------------
# 9. Unauthorized (no/invalid auth)
# ---------------------------------------------------------------------------
Test-Endpoint `
    -Name "9. GET /v1/catalog/products (no auth)" `
    -Url "$baseUrl/v1/catalog/products?limit=1" `
    -ExpectedStatus 401 `
    -Explanation "Protected endpoint without auth should return 401 Unauthorized."

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
Write-Host "`n========================================" -ForegroundColor DarkCyan
Write-Host " SUMMARY " -ForegroundColor Black -BackgroundColor $(if ($script:failCount -eq 0) { "Green" } else { "Yellow" })
Write-Host "========================================" -ForegroundColor DarkCyan
Write-Host "  Passed: $script:passCount" -ForegroundColor Green
Write-Host "  Failed: $script:failCount" -ForegroundColor $(if ($script:failCount -gt 0) { "Red" } else { "Green" })
if ($script:failCount -gt 0) {
    Write-Host "`n  Tip: Ensure OrderB2bAPI and ProductB2B are running." -ForegroundColor Yellow
    Write-Host "  Cart submit may fail (422) if SKU not in partner catalog." -ForegroundColor Yellow
}
Write-Host ""

# Test all OrderB2bAPI and ProductB2B endpoints
# Usage: .\scripts\test-endpoints.ps1
# Set these to match your environment:
$OrderB2bAPI = "http://localhost:8081"
$ProductB2B  = "http://localhost:3000"
$PartnerKey  = "42c54ce946a506c1dd581baabca141a6a0252eb55c1efffb07db5540eb6ec33b"
$ServiceKey  = "f7b7e8221adaa9330328c04df7df9bea80596a9511fb543600cb58a9689eecc2"

$AuthPartner = "Authorization: Bearer $PartnerKey"
$AuthService = "Authorization: Bearer $ServiceKey"

Write-Host "`n=== 1. OrderB2bAPI: Health (no auth) ===" -ForegroundColor Cyan
curl.exe -s "$OrderB2bAPI/health" | ConvertFrom-Json | ConvertTo-Json

Write-Host "`n=== 2. OrderB2bAPI: Catalog products (partner auth) ===" -ForegroundColor Cyan
$catalog = curl.exe -s -H $AuthPartner "$OrderB2bAPI/v1/catalog/products?limit=5"
$catalog | ConvertFrom-Json | ConvertTo-Json -Depth 5
# Capture first SKU for cart test if any
$catalogObj = $catalog | ConvertFrom-Json
$firstSku = $null
if ($catalogObj.data -and $catalogObj.data.Count -gt 0) {
    $firstSku = $catalogObj.data[0].sku
    Write-Host "First SKU from catalog: $firstSku" -ForegroundColor Green
}

Write-Host "`n=== 3. OrderB2bAPI: Cart submit (partner auth + idempotency) ===" -ForegroundColor Cyan
$idemKey = "test-cart-" + [guid]::NewGuid().ToString("N").Substring(0,12)
$cartPayload = @{
    partner_order_id = "test-order-" + (Get-Date -Format "yyyyMMdd-HHmmss")
    items = @(
        @{
            sku = if ($firstSku) { $firstSku } else { "TEST-SKU" }
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
try {
    $cartResp = Invoke-RestMethod -Uri "$OrderB2bAPI/v1/carts/submit" -Method POST -Headers @{ "Authorization" = "Bearer $PartnerKey"; "Idempotency-Key" = $idemKey } -ContentType "application/json" -Body ($cartPayload | ConvertTo-Json -Depth 5 -Compress)
    $cartResp | ConvertTo-Json -Depth 5
} catch {
    Write-Host "Response: $($_.Exception.Message)" -ForegroundColor Yellow
    if ($_.ErrorDetails.Message) { Write-Host $_.ErrorDetails.Message }
}

Write-Host "`n=== 4. OrderB2bAPI: List orders (partner auth) ===" -ForegroundColor Cyan
$listOrders = curl.exe -s -H $AuthPartner "$OrderB2bAPI/v1/admin/orders?limit=5"
$listOrders | ConvertFrom-Json | ConvertTo-Json -Depth 5
$ordersObj = $listOrders | ConvertFrom-Json
$firstOrderId = $null
if ($ordersObj.orders -and $ordersObj.orders.Count -gt 0) {
    $firstOrderId = $ordersObj.orders[0].id
    Write-Host "First order ID: $firstOrderId" -ForegroundColor Green
}

Write-Host "`n=== 5. OrderB2bAPI: Get order by ID (partner auth) ===" -ForegroundColor Cyan
$orderIdToGet = if ($firstOrderId) { $firstOrderId } else { "00000000-0000-0000-0000-000000000000" }
curl.exe -s -H $AuthPartner "$OrderB2bAPI/v1/orders/$orderIdToGet" | ConvertFrom-Json | ConvertTo-Json -Depth 5

Write-Host "`n=== 6. ProductB2B: Health (no auth) ===" -ForegroundColor Cyan
curl.exe -s "$ProductB2B/health"

Write-Host "`n=== 7. ProductB2B: Catalog by collection (service key) ===" -ForegroundColor Cyan
curl.exe -s -H $AuthService "$ProductB2B/v1/catalog/products?collection_handle=partner-zain&limit=3" | ConvertFrom-Json | ConvertTo-Json -Depth 4

Write-Host "`n=== 8. ProductB2B: Debug SKU lookup (no auth) ===" -ForegroundColor Cyan
curl.exe -s "$ProductB2B/debug/sku-lookup?sku=MK4820b" | ConvertFrom-Json | ConvertTo-Json

Write-Host "`n=== Done ===`n" -ForegroundColor Green

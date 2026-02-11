# Submit a test order (cart) to OrderB2bAPI
# Usage: .\scripts\test-order.ps1
# Ensure OrderB2bAPI is running and you have at least one product in the partner catalog.

$OrderB2bAPI = "http://localhost:8081"
$PartnerKey  = "42c54ce946a506c1dd581baabca141a6a0252eb55c1efffb07db5540eb6ec33b"

Write-Host "=== 1. Get catalog (to use a real SKU) ===" -ForegroundColor Cyan
$catalogJson = curl.exe -s -H "Authorization: Bearer $PartnerKey" "$OrderB2bAPI/v1/catalog/products?limit=5"
$catalog = $catalogJson | ConvertFrom-Json

$sku = $null
$title = "Test Product"
$price = 10.00

if ($catalog.data -and $catalog.data.Count -gt 0) {
    $first = $catalog.data[0]
    # Fallback response: { sku, title, price, image_url }
    if ($first.PSObject.Properties['sku']) {
        $sku = $first.sku
        $title = $first.title
        if ($first.price) { $price = [decimal]$first.price }
    }
    # ProductB2B response: product with variants.nodes[]
    elseif ($first.PSObject.Properties['variants']) {
        $title = $first.title
        $nodes = $null
        if ($first.variants.PSObject.Properties['nodes']) { $nodes = $first.variants.nodes }
        elseif ($first.variants -is [Array]) { $nodes = $first.variants }
        if ($nodes -and $nodes.Count -gt 0) {
            $v = $nodes[0]
            $sku = $v.sku
            if ($v.price) { $price = [decimal]$v.price }
        }
    }
}

if (-not $sku) {
    Write-Host "No SKU found in catalog. Using placeholder SKU - request may fail validation if SKU is not in partner catalog." -ForegroundColor Yellow
    $sku = "TEST-SKU"
}

Write-Host "Using SKU: $sku | Title: $title | Price: $price" -ForegroundColor Green

Write-Host "`n=== 2. Submit cart (create order) ===" -ForegroundColor Cyan
$partnerOrderId = "test-order-" + (Get-Date -Format "yyyyMMdd-HHmmss")
$idemKey = "idem-" + $partnerOrderId

$body = @{
    partner_order_id = $partnerOrderId
    items = @(
        @{
            sku = $sku
            title = $title
            price = $price
            quantity = 1
        }
    )
    customer = @{
        first_name = "Test"
        last_name = "Customer"
        email = "test@example.com"
        phone_number = "+962700000000"
    }
    shipping = @{
        city = "Amman"
        area = "Downtown"
        address = "123 Test Street"
        postal_code = "11181"
        country = "Jordan"
    }
    totals = @{
        subtotal = $price
        tax = 0
        shipping = 0
        total = $price
    }
}

$bodyJson = $body | ConvertTo-Json -Depth 5 -Compress
try {
    $response = Invoke-RestMethod -Uri "$OrderB2bAPI/v1/carts/submit" -Method POST `
        -Headers @{
            "Authorization" = "Bearer $PartnerKey"
            "Idempotency-Key" = $idemKey
            "Content-Type" = "application/json"
        } `
        -Body $bodyJson

    Write-Host "Order created successfully!" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 5
    Write-Host "`nGet order: curl -s -H `"Authorization: Bearer $PartnerKey`" `"$OrderB2bAPI/v1/orders/$($response.supplier_order_id)`"" -ForegroundColor Gray
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) { Write-Host $_.ErrorDetails.Message }
}

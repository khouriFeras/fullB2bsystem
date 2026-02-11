# Create a new order only. Add the order to Wassel manually, then run .\scripts\zain-delivery-status.ps1 <partner_order_id> to check delivery status.
# Usage: .\scripts\zain-order-and-delivery-status.ps1
# Uses partner API key (set $env:PARTNER_API_KEY or edit below). Ensure OrderB2bAPI is running.

$OrderB2bAPI = "http://localhost:8081"
$PartnerKey  = if ($env:PARTNER_API_KEY) { $env:PARTNER_API_KEY } else { "42c54ce946a506c1dd581baabca141a6a0252eb55c1efffb07db5540eb6ec33b" }

Write-Host "=== 1. Get catalog (to use a real SKU) ===" -ForegroundColor Cyan
try {
    $catalogJson = Invoke-RestMethod -Uri "$OrderB2bAPI/v1/catalog/products?limit=5" -Method GET -Headers @{ "Authorization" = "Bearer $PartnerKey" }
} catch {
    Write-Host "Failed to get catalog (check API key and server): $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$sku = "TEST-SKU"
$title = "Test Product"
$price = 10.00

if ($catalogJson.data -and $catalogJson.data.Count -gt 0) {
    $first = $catalogJson.data[0]
    if ($first.PSObject.Properties['sku']) {
        $sku = $first.sku
        $title = $first.title
        if ($first.price) { $price = [decimal]$first.price }
    } elseif ($first.PSObject.Properties['variants']) {
        $title = $first.title
        $nodes = $first.variants.nodes
        if (-not $nodes) { $nodes = $first.variants }
        if ($nodes -and $nodes.Count -gt 0) {
            $v = $nodes[0]
            $sku = $v.sku
            if ($v.price) { $price = [decimal]$v.price }
        }
    }
}
Write-Host "Using SKU: $sku | Price: $price" -ForegroundColor Green

Write-Host "`n=== 2. Create order (cart submit) ===" -ForegroundColor Cyan
$partnerOrderId = "zain-order-" + (Get-Date -Format "yyyyMMdd-HHmmss")
$idemKey = "idem-" + $partnerOrderId

# Delivery (shipping) address for the order
$deliveryAddress = @{
    city        = "Amman"
    area        = "Abdoun"
    address     = "123 Main St, Building 5"
    postal_code = "11181"
    country     = "Jordan"
}

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
        first_name = "Zain"
        last_name = "Customer"
        email = "zain@example.com"
        phone_number = "+962700000000"
    }
    shipping = $deliveryAddress
    totals = @{
        subtotal = $price
        tax = 0
        shipping = 0
        total = $price
    }
} | ConvertTo-Json -Depth 5 -Compress

$headers = @{
    "Authorization"   = "Bearer $PartnerKey"
    "Content-Type"    = "application/json"
    "Idempotency-Key" = $idemKey
}

try {
    $webResp = Invoke-WebRequest -Uri "$OrderB2bAPI/v1/carts/submit" -Method POST -Headers $headers -Body $body -UseBasicParsing
    if ($webResp.StatusCode -eq 204) {
        Write-Host "No order created (204). Cart had no partner catalog SKUs." -ForegroundColor Yellow
        Write-Host "Populate this partner's catalog first, then run this script again:" -ForegroundColor Yellow
        Write-Host "  go run cmd/map-partner-catalog/main.go partner-zain `"Partner Zain`"" -ForegroundColor Gray
        Write-Host "  (Use your collection handle and partner name; ensure the collection has products with SKUs.)" -ForegroundColor Gray
        exit 1
    }
    $createResp = $webResp.Content | ConvertFrom-Json
    Write-Host "Order created." -ForegroundColor Green
    $sid = if ($createResp.supplier_order_id) { $createResp.supplier_order_id } else { "(not returned)" }
    $poid = if ($createResp.partner_order_id) { $createResp.partner_order_id } else { $partnerOrderId }
    Write-Host "  supplier_order_id: $sid"
    Write-Host "  partner_order_id:  $poid"
    Write-Host "  Delivery address:  $($deliveryAddress.address), $($deliveryAddress.area), $($deliveryAddress.city) $($deliveryAddress.postal_code), $($deliveryAddress.country)"
    Write-Host ""
    Write-Host "Next: Add this order to Wassel manually (use partner_order_id as reference if needed)." -ForegroundColor Yellow
    Write-Host "Then run: .\scripts\zain-delivery-status.ps1 $poid" -ForegroundColor Gray
} catch {
    Write-Host "Cart submit failed: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) { Write-Host $_.ErrorDetails.Message }
    exit 1
}

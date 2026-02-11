# SKU Mapping Lifecycle Test (Simplified)
# Uses "Spong daddy" SKU which exists in Shopify but is not yet mapped

$BASE_URL = "http://localhost:8080"
$API_KEY = "test-api-key-123"
$testSKU = "Spong daddy"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "SKU Mapping Lifecycle Test" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Create order with unmapped SKU
Write-Host "Step 1: Create order with UNMAPPED SKU: $testSKU" -ForegroundColor Yellow
Write-Host "Expected: Order created, SKU appears as custom line item" -ForegroundColor Gray
Write-Host ""

$unmappedPayload = @{
    partner_order_id = "test-lifecycle-unmapped-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = $testSKU
            title = "Spong daddy Product"
            price = 3.5
            quantity = 1
            product_url = "https://example.com/spong-daddy"
        }
    )
    customer = @{
        name = "Lifecycle Test User"
        phone = "0772462582"
    }
    shipping = @{
        street = "Test Street"
        city = "Amman"
        state = "Khalda"
        postal_code = "00000"
        country = "JO"
    }
    totals = @{
        subtotal = 3.5
        tax = 0.35
        shipping = 100
        total = 103.85
    }
    payment_status = "paid"
    payment_method = "Cash On Delivery (COD)"
} | ConvertTo-Json -Depth 10

$headers = @{
    "Authorization" = "Bearer $API_KEY"
    "Content-Type" = "application/json"
    "Idempotency-Key" = [guid]::NewGuid().ToString()
}

try {
    $response1 = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers `
        -Body $unmappedPayload `
        -UseBasicParsing
    
    $responseBody1 = $response1.Content | ConvertFrom-Json
    $orderId1 = $responseBody1.supplier_order_id
    
    Write-Host "SUCCESS: Order created!" -ForegroundColor Green
    Write-Host "   Order ID: $orderId1" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Order creation failed!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Verify order 1
Write-Host "Verifying Order 1 (unmapped)..." -ForegroundColor Yellow
Start-Sleep -Seconds 1
$verify1 = go run cmd/check-order-items/main.go $orderId1 2>&1
$verify1 | Select-String -Pattern "Is Supplier Item|Shopify Variant ID"
Write-Host ""

# Step 2: Add SKU mapping
Write-Host "Step 2: Adding SKU mapping for $testSKU" -ForegroundColor Yellow
Write-Host "Product ID: 9039414001876, Variant ID: 47441132978388" -ForegroundColor Gray
Write-Host ""

$mappingResult = go run cmd/add-sku/main.go "Spong daddy" 9039414001876 47441132978388 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: SKU mapping added!" -ForegroundColor Green
} else {
    Write-Host "FAILURE: Could not add mapping (might already exist)" -ForegroundColor Yellow
    Write-Host $mappingResult
}
Write-Host ""

# Step 3: Create order again with same SKU (now mapped)
Write-Host "Step 3: Create order again with SAME SKU (now mapped)" -ForegroundColor Yellow
Write-Host "Expected: Order created, SKU now appears as variant line item" -ForegroundColor Gray
Write-Host ""

$mappedPayload = @{
    partner_order_id = "test-lifecycle-mapped-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = $testSKU
            title = "Spong daddy Product (Mapped)"
            price = 3.5
            quantity = 1
            product_url = "https://example.com/spong-daddy"
        }
    )
    customer = @{
        name = "Lifecycle Test User"
        phone = "0772462582"
    }
    shipping = @{
        street = "Test Street"
        city = "Amman"
        state = "Khalda"
        postal_code = "00000"
        country = "JO"
    }
    totals = @{
        subtotal = 3.5
        tax = 0.35
        shipping = 100
        total = 103.85
    }
    payment_status = "paid"
    payment_method = "Cash On Delivery (COD)"
} | ConvertTo-Json -Depth 10

$headers2 = @{
    "Authorization" = "Bearer $API_KEY"
    "Content-Type" = "application/json"
    "Idempotency-Key" = [guid]::NewGuid().ToString()
}

try {
    $response2 = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers2 `
        -Body $mappedPayload `
        -UseBasicParsing
    
    $responseBody2 = $response2.Content | ConvertFrom-Json
    $orderId2 = $responseBody2.supplier_order_id
    
    Write-Host "SUCCESS: Order created!" -ForegroundColor Green
    Write-Host "   Order ID: $orderId2" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Order creation failed!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Verify order 2
Write-Host "Verifying Order 2 (mapped)..." -ForegroundColor Yellow
Start-Sleep -Seconds 1
$verify2 = go run cmd/check-order-items/main.go $orderId2 2>&1
$verify2 | Select-String -Pattern "Is Supplier Item|Shopify Variant ID"
Write-Host ""

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "SKU Mapping Lifecycle Test Completed!" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Summary:" -ForegroundColor Yellow
Write-Host "  Order 1 (unmapped): $orderId1" -ForegroundColor Gray
Write-Host "    -> Should show: Is Supplier Item: false" -ForegroundColor Gray
Write-Host "  Order 2 (mapped):   $orderId2" -ForegroundColor Gray
Write-Host "    -> Should show: Is Supplier Item: true, Shopify Variant ID: 47441132978388" -ForegroundColor Gray
Write-Host ""

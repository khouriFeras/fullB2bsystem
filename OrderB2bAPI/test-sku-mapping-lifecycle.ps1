# SKU Mapping Lifecycle Test
# Tests:
# 1. Create order with unmapped SKU → appears as custom line item
# 2. Add SKU mapping
# 3. Create order again with same SKU → should now appear as variant line item

$BASE_URL = "http://localhost:8080"
$API_KEY = "test-api-key-123"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "SKU Mapping Lifecycle Test" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Create order with unmapped SKU
Write-Host "Step 1: Create order with UNMAPPED SKU" -ForegroundColor Yellow
Write-Host "Expected: Order created, SKU appears as custom line item" -ForegroundColor Gray
Write-Host ""

$testSKU = "LIFECYCLE-TEST-SKU-$(Get-Date -Format 'yyyyMMddHHmmss')"

$unmappedPayload = @{
    partner_order_id = "test-lifecycle-unmapped-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = $testSKU
            title = "Lifecycle Test Product"
            price = 15
            quantity = 1
            product_url = "https://example.com/lifecycle-test"
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
        subtotal = 15
        tax = 1.5
        shipping = 100
        total = 116.5
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
    Write-Host "   SKU: $testSKU (unmapped)" -ForegroundColor Gray
    Write-Host ""
    Write-Host "   Verify: go run cmd/check-order-items/main.go $orderId1" -ForegroundColor Gray
    Write-Host "   Expected: Is Supplier Item: false, Shopify Variant ID: (none)" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Order creation failed!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

# Step 2: Find the SKU in Shopify and add mapping
Write-Host "Step 2: Find SKU in Shopify and add mapping" -ForegroundColor Yellow
Write-Host "SKU to find: $testSKU" -ForegroundColor Gray
Write-Host ""
Write-Host "NOTE: This SKU doesn't exist in Shopify yet." -ForegroundColor Yellow
Write-Host "For this test to work, you need to:" -ForegroundColor Yellow
Write-Host "  1. Create a product in Shopify with SKU: $testSKU" -ForegroundColor Gray
Write-Host "  2. Get the Product ID and Variant ID" -ForegroundColor Gray
Write-Host "  3. Add the mapping using:" -ForegroundColor Gray
Write-Host "     go run cmd/add-sku/main.go `"$testSKU`" <product_id> <variant_id>" -ForegroundColor Gray
Write-Host ""
Write-Host "Press Enter after adding the mapping to continue..." -ForegroundColor Yellow
Read-Host

# Step 3: Create order again with same SKU (now mapped)
Write-Host "Step 3: Create order again with SAME SKU (now mapped)" -ForegroundColor Yellow
Write-Host "Expected: Order created, SKU now appears as variant line item" -ForegroundColor Gray
Write-Host ""

$mappedPayload = @{
    partner_order_id = "test-lifecycle-mapped-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = $testSKU
            title = "Lifecycle Test Product (Mapped)"
            price = 15
            quantity = 1
            product_url = "https://example.com/lifecycle-test"
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
        subtotal = 15
        tax = 1.5
        shipping = 100
        total = 116.5
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
    Write-Host "   SKU: $testSKU (now mapped)" -ForegroundColor Gray
    Write-Host ""
    Write-Host "   Verify: go run cmd/check-order-items/main.go $orderId2" -ForegroundColor Gray
    Write-Host "   Expected: Is Supplier Item: true, Shopify Variant ID: <variant_id>" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Order creation failed!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "SKU Mapping Lifecycle Test Completed!" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Summary:" -ForegroundColor Yellow
Write-Host "  Order 1 (unmapped): $orderId1" -ForegroundColor Gray
Write-Host "    -> Should show: Is Supplier Item: false" -ForegroundColor Gray
Write-Host "  Order 2 (mapped):   $orderId2" -ForegroundColor Gray
Write-Host "    -> Should show: Is Supplier Item: true" -ForegroundColor Gray
Write-Host ""

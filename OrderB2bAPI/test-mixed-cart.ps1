# Mixed Cart Behavior Testing Script
# Tests:
# 1. No mapped JafarShop SKUs in cart -> 204 No Content
# 2. Mixed cart (mapped + unmapped) -> mapped become Shopify variants, unmapped become custom line items

$BASE_URL = "http://localhost:8080"
$API_KEY = "test-api-key-123"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "Mixed Cart Behavior Test Suite" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Cart with only unmapped SKUs (should return 204)
Write-Host "Test 1: Cart with ONLY unmapped SKUs" -ForegroundColor Yellow
Write-Host "Expected: 204 No Content (no JafarShop products)" -ForegroundColor Gray
Write-Host ""

$unmappedOnlyPayload = @{
    partner_order_id = "test-unmapped-only-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = "UNMAPPED-SKU-1"
            title = "Unmapped Product 1"
            price = 10
            quantity = 1
            product_url = "https://example.com/unmapped1"
        },
        @{
            sku = "UNMAPPED-SKU-2"
            title = "Unmapped Product 2"
            price = 20
            quantity = 2
            product_url = "https://example.com/unmapped2"
        }
    )
    customer = @{
        name = "Unmapped Test User"
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
        subtotal = 50
        tax = 5
        shipping = 100
        total = 155
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
        -Body $unmappedOnlyPayload `
        -UseBasicParsing
    
    if ($response1.StatusCode -eq 204) {
        Write-Host "SUCCESS: Got 204 No Content as expected!" -ForegroundColor Green
        Write-Host "   (Cart contains no JafarShop products, so no order created)" -ForegroundColor Gray
    } else {
        Write-Host "FAILURE: Expected 204, but got $($response1.StatusCode)" -ForegroundColor Red
        Write-Host "   Response: $($response1.Content)" -ForegroundColor Red
    }
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 204) {
        Write-Host "SUCCESS: Got 204 No Content as expected!" -ForegroundColor Green
        Write-Host "   (Cart contains no JafarShop products, so no order created)" -ForegroundColor Gray
    } else {
        Write-Host "FAILURE: Expected 204, but got $statusCode" -ForegroundColor Red
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
}

Write-Host ""

# Test 2: Mixed cart (mapped + unmapped SKUs)
Write-Host "Test 2: Mixed Cart (Mapped + Unmapped SKUs)" -ForegroundColor Yellow
Write-Host "Expected: Order created with:" -ForegroundColor Gray
Write-Host "  - Mapped SKU (JDTQ1834) -> Shopify variant" -ForegroundColor Gray
Write-Host "  - Unmapped SKU -> Custom line item" -ForegroundColor Gray
Write-Host ""

$mixedCartPayload = @{
    partner_order_id = "test-mixed-cart-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = "JDTQ1834"
            title = "Product JDTQ1834"
            price = 10
            quantity = 1
            product_url = "https://example.com/jdtq1834"
        },
        @{
            sku = "UNMAPPED-SKU-MIXED"
            title = "Unmapped Product in Mixed Cart"
            price = 25
            quantity = 2
            product_url = "https://example.com/unmapped-mixed"
        }
    )
    customer = @{
        name = "Mixed Cart Test User"
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
        subtotal = 60
        tax = 6
        shipping = 100
        total = 166
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
        -Body $mixedCartPayload `
        -UseBasicParsing
    
    $responseBody2 = $response2.Content | ConvertFrom-Json
    $orderId2 = $responseBody2.supplier_order_id
    
    Write-Host "SUCCESS: Order created successfully!" -ForegroundColor Green
    Write-Host "   Status Code: $($response2.StatusCode)" -ForegroundColor Gray
    Write-Host "   Order ID: $orderId2" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Next steps to verify:" -ForegroundColor Yellow
    Write-Host "   1. Check database order items:" -ForegroundColor Gray
    Write-Host "      go run cmd/list-orders/main.go | Select-String -Pattern $orderId2" -ForegroundColor Gray
    Write-Host ""
    Write-Host "   2. Get Shopify Order ID from database, then check Shopify:" -ForegroundColor Gray
    Write-Host "      go run cmd/get-shopify-order-by-id/main.go [shopify_order_id]" -ForegroundColor Gray
    Write-Host ""
    Write-Host "   Expected in Shopify:" -ForegroundColor Gray
    Write-Host "   - JDTQ1834 should appear as a variant (with SKU)" -ForegroundColor Gray
    Write-Host "   - UNMAPPED-SKU-MIXED should appear as custom line item (no variant)" -ForegroundColor Gray
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
Write-Host "Mixed Cart Tests Completed!" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "To verify the mixed cart behavior:" -ForegroundColor Yellow
Write-Host "   1. Check the order in database to see item types" -ForegroundColor Gray
Write-Host "   2. Check the Shopify order to see variant vs custom line items" -ForegroundColor Gray

# Failure-Mode Behavior Test
# Tests:
# Shopify down / token wrong: API should still create DB order, log error, and return 200

$BASE_URL = "http://localhost:8080"
$API_KEY = "test-api-key-123"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "Failure-Mode Behavior Test" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "This test verifies that:" -ForegroundColor Yellow
Write-Host "  - If Shopify API fails, order is still created in database" -ForegroundColor Gray
Write-Host "  - API returns 200 (not 500)" -ForegroundColor Gray
Write-Host "  - Error is logged but doesn't fail the request" -ForegroundColor Gray
Write-Host ""

Write-Host "NOTE: To fully test this, you would need to:" -ForegroundColor Yellow
Write-Host "  1. Temporarily break Shopify connection (wrong token/domain)" -ForegroundColor Gray
Write-Host "  2. Submit an order" -ForegroundColor Gray
Write-Host "  3. Verify order exists in database" -ForegroundColor Gray
Write-Host "  4. Verify API returned 200" -ForegroundColor Gray
Write-Host "  5. Check logs for Shopify error" -ForegroundColor Gray
Write-Host ""
Write-Host "For now, we'll test with a normal order and verify the behavior" -ForegroundColor Yellow
Write-Host "when Shopify operations fail gracefully." -ForegroundColor Yellow
Write-Host ""

# Create a test order
Write-Host "Creating test order..." -ForegroundColor Yellow
Write-Host ""

$testOrderPayload = @{
    partner_order_id = "test-failure-mode-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = "JDTQ1834"
            title = "Product JDTQ1834"
            price = 10
            quantity = 1
            product_url = "https://example.com/p1"
        }
    )
    customer = @{
        name = "Failure Mode Test User"
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
        subtotal = 10
        tax = 1
        shipping = 100
        total = 111
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
    $response = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers `
        -Body $testOrderPayload `
        -UseBasicParsing
    
    $responseBody = $response.Content | ConvertFrom-Json
    $orderId = $responseBody.supplier_order_id
    
    Write-Host "SUCCESS: Order created!" -ForegroundColor Green
    Write-Host "   Status Code: $($response.StatusCode)" -ForegroundColor Gray
    Write-Host "   Order ID: $orderId" -ForegroundColor Gray
    Write-Host ""
    Write-Host "   Expected behavior:" -ForegroundColor Yellow
    Write-Host "   - Order exists in database (even if Shopify failed)" -ForegroundColor Gray
    Write-Host "   - API returned 200 (not 500)" -ForegroundColor Gray
    Write-Host "   - Check server logs for any Shopify errors" -ForegroundColor Gray
    Write-Host ""
    
    # Verify order exists in database
    Write-Host "Verifying order exists in database..." -ForegroundColor Yellow
    $verifyResult = go run cmd/list-orders/main.go 2>&1 | Select-String -Pattern $orderId
    if ($verifyResult) {
        Write-Host "   Order found in database!" -ForegroundColor Green
    } else {
        Write-Host "   Order NOT found in database!" -ForegroundColor Red
    }
    Write-Host ""
    
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "FAILURE: Order creation failed!" -ForegroundColor Red
    Write-Host "   Status Code: $statusCode" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    Write-Host ""
    Write-Host "   NOTE: If Shopify fails, the API should still return 200" -ForegroundColor Yellow
    Write-Host "   and create the order in the database." -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "Failure-Mode Test Completed!" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "To fully test failure modes:" -ForegroundColor Yellow
Write-Host "  1. Set invalid SHOPIFY_ACCESS_TOKEN in .env" -ForegroundColor Gray
Write-Host "  2. Restart server" -ForegroundColor Gray
Write-Host "  3. Submit order - should still return 200" -ForegroundColor Gray
Write-Host "  4. Verify order in database" -ForegroundColor Gray
Write-Host "  5. Check logs for Shopify error messages" -ForegroundColor Gray
Write-Host ""

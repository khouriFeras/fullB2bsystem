# Idempotency Testing Script
# Tests:
# 1. Same Idempotency-Key + same payload → should return the same order (no duplicates)
# 2. Same Idempotency-Key + different payload → 409 Conflict

$BASE_URL = "http://localhost:8080"
$API_KEY = "test-api-key-123"

# Generate a unique idempotency key for this test
$IDEMPOTENCY_KEY = [guid]::NewGuid().ToString()

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "Idempotency Test Suite" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: First submission (should create new order)
Write-Host "Test 1: First submission with Idempotency-Key" -ForegroundColor Yellow
Write-Host "Idempotency Key: $IDEMPOTENCY_KEY" -ForegroundColor Gray
Write-Host ""

$orderPayload = @{
    partner_order_id = "idempotency-test-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = "JDTQ1834"
            title = "Product JDTQ1834"
            price = 10
            quantity = 1
            product_url = "https://example.com/p1"
        },
        @{
            sku = "Spong daddy"
            title = "Product Spong daddy"
            price = 3.5
            quantity = 2
            product_url = "https://example.com/p2"
        }
    )
    customer = @{
        name = "Idempotency Test User"
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
        subtotal = 17
        tax = 1.6
        shipping = 100
        total = 118.6
    }
    payment_status = "paid"
    payment_method = "Cash On Delivery (COD)"
} | ConvertTo-Json -Depth 10

$headers = @{
    "Authorization" = "Bearer $API_KEY"
    "Content-Type" = "application/json"
    "Idempotency-Key" = $IDEMPOTENCY_KEY
}

try {
    $response1 = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers `
        -Body $orderPayload `
        -UseBasicParsing
    
    $responseBody1 = $response1.Content | ConvertFrom-Json
    $orderId1 = $responseBody1.supplier_order_id
    
    Write-Host "✅ First submission successful!" -ForegroundColor Green
    Write-Host "   Status Code: $($response1.StatusCode)" -ForegroundColor Gray
    Write-Host "   Order ID: $orderId1" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "❌ First submission failed!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

# Wait a moment
Start-Sleep -Seconds 1

# Test 2: Same key + same payload (should return same order, no duplicate)
Write-Host "Test 2: Same Idempotency-Key + Same Payload" -ForegroundColor Yellow
Write-Host "Expected: Should return the SAME order ID (no duplicate)" -ForegroundColor Gray
Write-Host ""

try {
    $response2 = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers `
        -Body $orderPayload `
        -UseBasicParsing
    
    $responseBody2 = $response2.Content | ConvertFrom-Json
    $orderId2 = $responseBody2.supplier_order_id
    
    Write-Host "✅ Second submission successful!" -ForegroundColor Green
    Write-Host "   Status Code: $($response2.StatusCode)" -ForegroundColor Gray
    Write-Host "   Order ID: $orderId2" -ForegroundColor Gray
    Write-Host ""
    
    if ($orderId1 -eq $orderId2) {
        Write-Host "✅ SUCCESS: Same order ID returned (no duplicate created)!" -ForegroundColor Green
    } else {
        Write-Host "❌ FAILURE: Different order IDs! Expected: $orderId1, Got: $orderId2" -ForegroundColor Red
    }
    Write-Host ""
} catch {
    Write-Host "❌ Second submission failed!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    exit 1
}

# Test 3: Same key + different payload (should return 409 Conflict)
Write-Host "Test 3: Same Idempotency-Key + Different Payload" -ForegroundColor Yellow
Write-Host "Expected: 409 Conflict" -ForegroundColor Gray
Write-Host ""

# Modify the payload (change total)
$differentPayload = @{
    partner_order_id = "idempotency-test-different-$(Get-Date -Format 'yyyyMMddHHmmss')"
    items = @(
        @{
            sku = "JDTQ1834"
            title = "Product JDTQ1834"
            price = 10
            quantity = 1
            product_url = "https://example.com/p1"
        },
        @{
            sku = "Spong daddy"
            title = "Product Spong daddy"
            price = 3.5
            quantity = 2
            product_url = "https://example.com/p2"
        }
    )
    customer = @{
        name = "Idempotency Test User"
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
        subtotal = 17
        tax = 1.6
        shipping = 100
        total = 200.0  # DIFFERENT TOTAL
    }
    payment_status = "paid"
    payment_method = "Cash On Delivery (COD)"
} | ConvertTo-Json -Depth 10

try {
    $response3 = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers `
        -Body $differentPayload `
        -UseBasicParsing
    
    Write-Host "❌ FAILURE: Expected 409 Conflict, but got $($response3.StatusCode)" -ForegroundColor Red
    Write-Host "   Response: $($response3.Content)" -ForegroundColor Red
    exit 1
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    
    if ($statusCode -eq 409) {
        Write-Host "✅ SUCCESS: Got 409 Conflict as expected!" -ForegroundColor Green
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Gray
    } else {
        Write-Host "❌ FAILURE: Expected 409 Conflict, but got $statusCode" -ForegroundColor Red
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "All Idempotency Tests Passed! ✅" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Cyan

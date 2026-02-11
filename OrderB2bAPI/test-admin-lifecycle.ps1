# Admin Order Lifecycle Test
# Tests:
# 1. POST /v1/admin/orders/:id/confirm
# 2. POST /v1/admin/orders/:id/reject (with reason)
# 3. POST /v1/admin/orders/:id/ship (tracking info)
# 4. Verify GET /v1/orders/:id reflects status transitions

$BASE_URL = "http://localhost:8080"
$API_KEY = "test-api-key-123"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "Admin Order Lifecycle Test Suite" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# First, create an order to work with
Write-Host "Step 0: Creating test order..." -ForegroundColor Yellow
Write-Host ""

$testOrderPayload = @{
    partner_order_id = "test-admin-lifecycle-$(Get-Date -Format 'yyyyMMddHHmmss')"
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
        name = "Admin Test User"
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
    
    Write-Host "SUCCESS: Test order created!" -ForegroundColor Green
    Write-Host "   Order ID: $orderId" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Could not create test order!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Check initial status
Write-Host "Initial Order Status:" -ForegroundColor Yellow
try {
    $statusResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/orders/$orderId" `
        -Method GET `
        -Headers @{ "Authorization" = "Bearer $API_KEY" } `
        -UseBasicParsing
    
    $statusBody = $statusResponse.Content | ConvertFrom-Json
    Write-Host "   Status: $($statusBody.status)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "   Could not get initial status" -ForegroundColor Yellow
    Write-Host ""
}

# Test 1: Confirm Order
Write-Host "Test 1: POST /v1/admin/orders/:id/confirm" -ForegroundColor Yellow
Write-Host "Expected: Status changes to CONFIRMED" -ForegroundColor Gray
Write-Host ""

try {
    $confirmResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/admin/orders/$orderId/confirm" `
        -Method POST `
        -Headers @{ "Authorization" = "Bearer $API_KEY" } `
        -UseBasicParsing
    
    $confirmBody = $confirmResponse.Content | ConvertFrom-Json
    
    Write-Host "SUCCESS: Order confirmed!" -ForegroundColor Green
    Write-Host "   Status Code: $($confirmResponse.StatusCode)" -ForegroundColor Gray
    Write-Host "   New Status: $($confirmBody.status)" -ForegroundColor Gray
    Write-Host ""
    
    if ($confirmBody.status -eq "CONFIRMED") {
        Write-Host "   Status transition verified!" -ForegroundColor Green
    } else {
        Write-Host "   WARNING: Expected CONFIRMED, got $($confirmBody.status)" -ForegroundColor Yellow
    }
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Could not confirm order!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    Write-Host ""
}

# Verify status via GET
Write-Host "Verifying status via GET /v1/orders/:id..." -ForegroundColor Yellow
try {
    $verifyResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/orders/$orderId" `
        -Method GET `
        -Headers @{ "Authorization" = "Bearer $API_KEY" } `
        -UseBasicParsing
    
    $verifyBody = $verifyResponse.Content | ConvertFrom-Json
    Write-Host "   Status: $($verifyBody.status)" -ForegroundColor Gray
    Write-Host ""
} catch {
    Write-Host "   Could not verify status" -ForegroundColor Yellow
    Write-Host ""
}

# Test 2: Reject Order
Write-Host "Test 2: POST /v1/admin/orders/:id/reject (with reason)" -ForegroundColor Yellow
Write-Host "Expected: Status changes to REJECTED with rejection_reason" -ForegroundColor Gray
Write-Host ""

# Create another order for rejection test
$rejectOrderPayload = @{
    partner_order_id = "test-admin-reject-$(Get-Date -Format 'yyyyMMddHHmmss')"
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
        name = "Reject Test User"
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

$headers2 = @{
    "Authorization" = "Bearer $API_KEY"
    "Content-Type" = "application/json"
    "Idempotency-Key" = [guid]::NewGuid().ToString()
}

try {
    $rejectOrderResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" `
        -Method POST `
        -Headers $headers2 `
        -Body $rejectOrderPayload `
        -UseBasicParsing
    
    $rejectOrderBody = $rejectOrderResponse.Content | ConvertFrom-Json
    $rejectOrderId = $rejectOrderBody.supplier_order_id
    
    Write-Host "   Created order for rejection test: $rejectOrderId" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "   Could not create order for rejection test" -ForegroundColor Yellow
    Write-Host ""
}

if ($rejectOrderId) {
    $rejectPayload = @{
        reason = "Product out of stock"
    } | ConvertTo-Json

    try {
        $rejectResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/admin/orders/$rejectOrderId/reject" `
            -Method POST `
            -Headers @{
                "Authorization" = "Bearer $API_KEY"
                "Content-Type" = "application/json"
            } `
            -Body $rejectPayload `
            -UseBasicParsing
        
        $rejectBody = $rejectResponse.Content | ConvertFrom-Json
        
        Write-Host "SUCCESS: Order rejected!" -ForegroundColor Green
        Write-Host "   Status Code: $($rejectResponse.StatusCode)" -ForegroundColor Gray
        Write-Host "   New Status: $($rejectBody.status)" -ForegroundColor Gray
        Write-Host "   Rejection Reason: $($rejectBody.rejection_reason)" -ForegroundColor Gray
        Write-Host ""
        
        if ($rejectBody.status -eq "REJECTED") {
            Write-Host "   Status transition verified!" -ForegroundColor Green
        } else {
            Write-Host "   WARNING: Expected REJECTED, got $($rejectBody.status)" -ForegroundColor Yellow
        }
        Write-Host ""
        
    } catch {
        Write-Host "FAILURE: Could not reject order!" -ForegroundColor Red
        Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $responseBody = $reader.ReadToEnd()
            Write-Host "   Response: $responseBody" -ForegroundColor Red
        }
        Write-Host ""
    }
}

# Test 3: Ship Order
Write-Host "Test 3: POST /v1/admin/orders/:id/ship (tracking info)" -ForegroundColor Yellow
Write-Host "Expected: Status changes to SHIPPED with tracking info" -ForegroundColor Gray
Write-Host ""

# Use the confirmed order from Test 1
$shipPayload = @{
    carrier = "Standard Shipping"
    tracking_number = "TRACK123456789"
    tracking_url = "https://example.com/track/TRACK123456789"
} | ConvertTo-Json

try {
    $shipResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/admin/orders/$orderId/ship" `
        -Method POST `
        -Headers @{
            "Authorization" = "Bearer $API_KEY"
            "Content-Type" = "application/json"
        } `
        -Body $shipPayload `
        -UseBasicParsing
    
    $shipBody = $shipResponse.Content | ConvertFrom-Json
    
    Write-Host "SUCCESS: Order shipped!" -ForegroundColor Green
    Write-Host "   Status Code: $($shipResponse.StatusCode)" -ForegroundColor Gray
    Write-Host "   New Status: $($shipBody.status)" -ForegroundColor Gray
    Write-Host "   Tracking Carrier: $($shipBody.tracking_carrier)" -ForegroundColor Gray
    Write-Host "   Tracking Number: $($shipBody.tracking_number)" -ForegroundColor Gray
    Write-Host ""
    
    if ($shipBody.status -eq "SHIPPED") {
        Write-Host "   Status transition verified!" -ForegroundColor Green
    } else {
        Write-Host "   WARNING: Expected SHIPPED, got $($shipBody.status)" -ForegroundColor Yellow
    }
    Write-Host ""
    
} catch {
    Write-Host "FAILURE: Could not ship order!" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "   Response: $responseBody" -ForegroundColor Red
    }
    Write-Host ""
}

# Final verification via GET
Write-Host "Final Status Verification via GET /v1/orders/:id..." -ForegroundColor Yellow
try {
    $finalResponse = Invoke-WebRequest -Uri "$BASE_URL/v1/orders/$orderId" `
        -Method GET `
        -Headers @{ "Authorization" = "Bearer $API_KEY" } `
        -UseBasicParsing
    
    $finalBody = $finalResponse.Content | ConvertFrom-Json
    
    Write-Host "   Final Status: $($finalBody.status)" -ForegroundColor Gray
    Write-Host "   Tracking Carrier: $($finalBody.tracking_carrier)" -ForegroundColor Gray
    Write-Host "   Tracking Number: $($finalBody.tracking_number)" -ForegroundColor Gray
    Write-Host "   Tracking URL: $($finalBody.tracking_url)" -ForegroundColor Gray
    Write-Host ""
    
} catch {
    Write-Host "   Could not get final status" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "Admin Order Lifecycle Tests Completed!" -ForegroundColor Green
Write-Host "================================================" -ForegroundColor Cyan

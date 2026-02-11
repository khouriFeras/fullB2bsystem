# Test Order Submission Script with Mapped SKU
$API_KEY = "test-api-key-123"
$BASE_URL = "http://localhost:8081"
$IdempotencyKey = [System.Guid]::NewGuid().ToString()

Write-Host "Testing Order Submission with Mapped SKU" -ForegroundColor Cyan
Write-Host "API Key: $API_KEY" -ForegroundColor Yellow
Write-Host "Idempotency Key: $IdempotencyKey" -ForegroundColor Yellow
Write-Host ""

# Use the test order with mapped SKU
$orderJson = Get-Content -Path "test-order-with-mapped-sku.json" -Raw

$headers = @{
    "Authorization" = "Bearer $API_KEY"
    "Idempotency-Key" = $IdempotencyKey
    "Content-Type" = "application/json"
}

Write-Host "Submitting order..." -ForegroundColor Green
Write-Host "Note: This order includes:" -ForegroundColor Gray
Write-Host "  - Drinkmate-Syrup3 (mapped SKU from Shopify)" -ForegroundColor Green
Write-Host "  - Spong daddy (custom product)" -ForegroundColor Yellow
Write-Host "  - notInOurSotre (custom product)" -ForegroundColor Yellow
Write-Host ""

try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/v1/carts/submit" -Method POST -Headers $headers -Body $orderJson -UseBasicParsing -ErrorAction Stop

    Write-Host "Success! Order created" -ForegroundColor Green
    Write-Host "Response Status: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "Response Body:" -ForegroundColor Green
    $response.Content | ConvertFrom-Json | ConvertTo-Json -Depth 10
    
    $orderData = $response.Content | ConvertFrom-Json
    if ($orderData.supplier_order_id) {
        Write-Host ""
        Write-Host "Order ID: $($orderData.supplier_order_id)" -ForegroundColor Cyan
        Write-Host "Status: $($orderData.status)" -ForegroundColor Cyan
    }
}
catch {
    Write-Host "Error occurred!" -ForegroundColor Red
    Write-Host ""
    
    if ($_.Exception.Response) {
        $statusCode = $_.Exception.Response.StatusCode.value__
        Write-Host "Status Code: $statusCode" -ForegroundColor Red
        
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        $reader.Close()
        
        if ($responseBody) {
            Write-Host "Error Response:" -ForegroundColor Red
            try {
                $errorData = $responseBody | ConvertFrom-Json
                $errorData | ConvertTo-Json -Depth 10
            }
            catch {
                Write-Host $responseBody
            }
        }
        
        Write-Host ""
        Write-Host "Common Issues:" -ForegroundColor Yellow
        Write-Host "  - 401: Check your API key is correct" -ForegroundColor Gray
        Write-Host "  - 204: No supplier SKUs found (all SKUs unmapped)" -ForegroundColor Gray
        Write-Host "  - 422: Check the JSON format and required fields" -ForegroundColor Gray
    }
    else {
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

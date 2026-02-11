# Test Order Submission Script
# Use your partner API key (from create-partner). Replace with your key or set env: $env:PARTNER_API_KEY = "your-key"
$API_KEY = if ($env:PARTNER_API_KEY) { $env:PARTNER_API_KEY } else { "42c54ce946a506c1dd581baabca141a6a0252eb55c1efffb07db5540eb6ec33b" }
$BASE_URL = "http://localhost:8081"
$IdempotencyKey = [System.Guid]::NewGuid().ToString()

Write-Host "Testing Order Submission" -ForegroundColor Cyan
Write-Host "API Key: $API_KEY" -ForegroundColor Yellow
Write-Host "Idempotency Key: $IdempotencyKey" -ForegroundColor Yellow
Write-Host ""

# Read JSON as UTF-8 (handles Arabic and strips BOM if present)
$orderJson = Get-Content -Path "test-order.json" -Raw -Encoding UTF8
if ($orderJson.Length -gt 3 -and $orderJson[0] -eq [char]0xFEFF) { $orderJson = $orderJson.Substring(1) }

$headers = @{
    "Authorization" = "Bearer $API_KEY"
    "Idempotency-Key" = $IdempotencyKey
    "Content-Type" = "application/json; charset=utf-8"
}

Write-Host "Submitting order..." -ForegroundColor Green
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
        if ($orderData.shopify_order_id) {
            Write-Host "Shopify Order ID: $($orderData.shopify_order_id)" -ForegroundColor Green
        }
        if ($orderData.shopify_draft_order_id) {
            Write-Host "Shopify Draft Order ID: $($orderData.shopify_draft_order_id)" -ForegroundColor Gray
        }
        if ($orderData.shopify_error) {
            Write-Host ""
            Write-Host "Shopify Error (order not created in Shopify): $($orderData.shopify_error)" -ForegroundColor Red
        }
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
            Write-Host $responseBody -ForegroundColor Red
            try {
                $errorData = $responseBody | ConvertFrom-Json
                Write-Host ""
                Write-Host "Parsed Error:" -ForegroundColor Yellow
                $errorData | ConvertTo-Json -Depth 10
                if ($statusCode -eq 422 -and $errorData.details) {
                    Write-Host ""
                    Write-Host "Validation error (fix the payload or required fields):" -ForegroundColor Yellow
                    Write-Host $errorData.details -ForegroundColor White
                }
            }
            catch {
                # Already displayed raw response
            }
        }
        
        Write-Host ""
        Write-Host "Common Issues:" -ForegroundColor Yellow
        Write-Host "  - 401: Check your API key is correct (create partner first)" -ForegroundColor Gray
        Write-Host "  - 204: SKU is not mapped. This is expected if SKU doesn't exist in database." -ForegroundColor Gray
        Write-Host "  - 422: Check the JSON format and required fields (see 'Validation error' above)" -ForegroundColor Gray
        Write-Host "  - 500: Check server logs, might be missing partner or database issue" -ForegroundColor Gray
        Write-Host ""
        Write-Host "To create a partner, run:" -ForegroundColor Yellow
        Write-Host "  go run cmd/create-partner/main.go `"Test Partner`" `"test-api-key-123`"" -ForegroundColor Gray
    }
    else {
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

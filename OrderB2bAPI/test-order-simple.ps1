# Simple test to capture full error response
$API_KEY = "test-api-key-123"
$BASE_URL = "http://localhost:8081"

$orderJson = @"
{
  "partner_order_id": "test-order-127",
  "items": [
    {
      "sku": "JDTQ1834",
      "title": "Product JDTQ1834",
      "price": 10,
      "quantity": 1,
      "product_url": "https://example.com/p1"
    }
  ],
  "customer": {
    "name": "test user",
    "phone": "0772462582"
  },
  "shipping": {
    "street": "na",
    "city": "amman",
    "state": "khalda",
    "postal_code": "00000",
    "country": "JO"
  },
  "totals": {
    "subtotal": 10,
    "tax": 0,
    "shipping": 0,
    "total": 10
  },
  "payment_status": "paid",
  "payment_method": "Cash On Delivery (COD)"
}
"@

$headers = @{
    "Authorization" = "Bearer $API_KEY"
    "Idempotency-Key" = [System.Guid]::NewGuid().ToString()
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/v1/carts/submit" -Method POST -Headers $headers -Body $orderJson -ErrorAction Stop
    Write-Host "Success!" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Error Status: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Red
    
    $stream = $_.Exception.Response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($stream)
    $responseBody = $reader.ReadToEnd()
    $reader.Close()
    $stream.Close()
    
    Write-Host "Error Body:" -ForegroundColor Red
    Write-Host $responseBody -ForegroundColor Red
}

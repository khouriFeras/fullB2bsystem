param(
    [Parameter(Mandatory=$true)]
    [string]$OrderID
)

$API_KEY = "test-api-key-123"
$BASE_URL = "http://localhost:8081"

Write-Host "Checking order status..." -ForegroundColor Cyan
Write-Host "Order ID: $OrderID"
Write-Host ""

$headers = @{
    "Authorization" = "Bearer $API_KEY"
    "Content-Type" = "application/json"
}

try {
    $response = Invoke-RestMethod -Uri "$BASE_URL/v1/orders/$OrderID" -Method GET -Headers $headers
    
    Write-Host "=== Order Found ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "Order ID:          $($response.id)"
    Write-Host "Partner Order ID:  $($response.partner_order_id)"
    Write-Host "Status:            $($response.status)" -ForegroundColor Yellow
    Write-Host "Customer:          $($response.customer_name)"
    
    if ($response.customer_phone) {
        Write-Host "Phone:             $($response.customer_phone)"
    }
    
    Write-Host "Cart Total:        `$$($response.cart_total)"
    Write-Host "Payment Status:    $($response.payment_status)"
    Write-Host "Payment Method:    $($response.payment_method)"
    
    if ($response.shopify_draft_order_id) {
        Write-Host "Shopify Draft ID:  $($response.shopify_draft_order_id)" -ForegroundColor Cyan
    }
    
    if ($response.shopify_order_id) {
        Write-Host "Shopify Order ID:  $($response.shopify_order_id)" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host "Items ($($response.items.Count)):" -ForegroundColor Yellow
    
    foreach ($item in $response.items) {
        $badge = if ($item.is_supplier_item) { "[Supplier]" } else { "[Custom]" }
        $total = [math]::Round($item.quantity * $item.price, 2)
        Write-Host "  - $($item.sku): $($item.title)"
        Write-Host "    Qty: $($item.quantity) x `$$($item.price) = `$$total $badge"
    }
    
    Write-Host ""
    Write-Host "Created: $($response.created_at)" -ForegroundColor Gray
    Write-Host "Updated: $($response.updated_at)" -ForegroundColor Gray
    
} catch {
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
}

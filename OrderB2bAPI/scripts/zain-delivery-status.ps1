# Check delivery status for an order (run after adding the order to Wassel manually).
# Usage: .\scripts\zain-delivery-status.ps1 <partner_order_id_or_supplier_uuid>
# Example: .\scripts\zain-delivery-status.ps1 zain-order-20260209120000

param(
    [Parameter(Mandatory = $true)]
    [string]$OrderId
)

$OrderB2bAPI = "http://localhost:8081"
$PartnerKey  = if ($env:PARTNER_API_KEY) { $env:PARTNER_API_KEY } else { "42c54ce946a506c1dd581baabca141a6a0252eb55c1efffb07db5540eb6ec33b" }

Write-Host "=== Delivery status for order: $OrderId ===" -ForegroundColor Cyan
try {
    $resp = Invoke-RestMethod -Uri "$OrderB2bAPI/v1/orders/$OrderId/delivery-status" -Method GET -Headers @{ "Authorization" = "Bearer $PartnerKey" }
    Write-Host "Response:" -ForegroundColor Green
    $resp | ConvertTo-Json -Depth 10
} catch {
    Write-Host "Request failed: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails.Message) { Write-Host $_.ErrorDetails.Message }
    exit 1
}

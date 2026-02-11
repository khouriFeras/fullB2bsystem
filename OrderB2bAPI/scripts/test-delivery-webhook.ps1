# Test delivery webhook: run a local HTTP listener, then trigger the API so it POSTs to this listener.
# Prereqs: OrderB2bAPI and GetDeliveryStatus running; a partner with an order that has delivery data (or use ship).
#
# Usage:
#   1. In this terminal: .\scripts\test-delivery-webhook.ps1
#   2. In another terminal:
#      - Set webhook URL (use the URL printed below, e.g. http://localhost:9999):
#        go run cmd/set-partner-webhook/main.go --partner-id <PARTNER_UUID> --webhook-url http://localhost:9999
#      - Trigger delivery-status (or ship an order):
#        .\scripts\zain-delivery-status.ps1 <partner_order_id>
#        OR ship: Invoke-RestMethod -Uri "http://localhost:8081/v1/admin/orders/<id>/ship" -Method POST -Headers @{ "Authorization" = "Bearer $env:PARTNER_API_KEY" } -ContentType "application/json" -Body '{"carrier":"Wassel","tracking_number":"123"}'
#   3. This script will print the received JSON and exit.

$Port = 9999
$Prefix = "http://localhost:$Port/"

$Listener = New-Object System.Net.HttpListener
$Listener.Prefixes.Add($Prefix)
$Listener.Start()
Write-Host "=== Delivery webhook test listener ===" -ForegroundColor Cyan
Write-Host "Listening on $Prefix" -ForegroundColor Green
Write-Host ""
Write-Host "Set partner webhook and trigger the API:" -ForegroundColor Yellow
Write-Host "  go run cmd/set-partner-webhook/main.go --partner-id <PARTNER_UUID> --webhook-url $Prefix" -ForegroundColor Gray
Write-Host "  .\scripts\zain-delivery-status.ps1 <partner_order_id>" -ForegroundColor Gray
Write-Host ""
Write-Host "Waiting for POST..." -ForegroundColor Cyan

$Context = $Listener.GetContext()
$Request = $Context.Request
$Response = $Context.Response

if ($Request.HttpMethod -ne "POST") {
    $Response.StatusCode = 405
    $Response.Close()
    Write-Host "Received $($Request.HttpMethod), expected POST. Ignored." -ForegroundColor Red
    $Listener.Stop()
    exit 0
}

$Reader = New-Object System.IO.StreamReader($Request.InputStream, $Request.ContentEncoding)
$Body = $Reader.ReadToEnd()
$Reader.Close()

$Response.StatusCode = 200
$Response.ContentType = "application/json"
$Response.OutputStream.Write([System.Text.Encoding]::UTF8.GetBytes("{}"), 0, 2)
$Response.OutputStream.Close()
$Response.Close()

$Listener.Stop()

Write-Host "Received webhook POST:" -ForegroundColor Green
try {
    $Json = $Body | ConvertFrom-Json
    $Body | ConvertTo-Json -Depth 20
} catch {
    $Body
}

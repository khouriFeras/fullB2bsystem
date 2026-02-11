# Script to setup test partner and verify everything is ready

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Test Partner Setup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$PARTNER_NAME = "Test Partner"
$API_KEY = "test-api-key-123"

Write-Host "Creating partner..." -ForegroundColor Yellow
Write-Host "  Name: $PARTNER_NAME" -ForegroundColor Gray
Write-Host "  API Key: $API_KEY" -ForegroundColor Gray
Write-Host ""

$output = & go run cmd/create-partner/main.go $PARTNER_NAME $API_KEY 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host $output -ForegroundColor Green
    Write-Host ""
    Write-Host "✅ Partner created successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "You can now test order submission with:" -ForegroundColor Yellow
    Write-Host "  .\test-order.ps1" -ForegroundColor Gray
} else {
    Write-Host $output -ForegroundColor Red
    Write-Host ""
    Write-Host "⚠️  Partner creation failed. It might already exist." -ForegroundColor Yellow
    Write-Host "   If the partner already exists, you can proceed with testing." -ForegroundColor Gray
}

Write-Host ""

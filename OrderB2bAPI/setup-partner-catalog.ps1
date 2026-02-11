# Script to clear SKU mappings and map all products from "Partner Catalog" collection

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Partner Catalog SKU Mapping Setup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "This script will:" -ForegroundColor Yellow
Write-Host "  1. Clear all existing SKU mappings from the database" -ForegroundColor Gray
Write-Host "  2. Map all products from 'Partner Catalog' collection" -ForegroundColor Gray
Write-Host ""

$confirm = Read-Host "Continue? (yes/no)"
if ($confirm -ne "yes") {
    Write-Host "Operation cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "[1/2] Clearing existing SKU mappings..." -ForegroundColor Yellow
Write-Host ""

# Clear mappings (will prompt for confirmation)
$clearOutput = & go run cmd/clear-sku-mappings/main.go 2>&1
Write-Host $clearOutput

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "❌ Failed to clear SKU mappings" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[2/2] Mapping products from 'Partner Catalog' collection..." -ForegroundColor Yellow
Write-Host ""

# Map products from collection
$mapOutput = & go run cmd/map-partner-catalog/main.go 2>&1
Write-Host $mapOutput

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  Setup Complete!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "  - Verify mappings: go run cmd/list-sku-mappings/main.go" -ForegroundColor Gray
    Write-Host "  - Test order submission: .\test-order.ps1" -ForegroundColor Gray
    Write-Host ""
} else {
    Write-Host ""
    Write-Host "❌ Failed to map products" -ForegroundColor Red
    exit 1
}

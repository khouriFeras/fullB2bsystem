# Script to map SKUs from test-order.json to Shopify products
# This will find the SKUs in Shopify and create mappings

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Map Test Order SKUs to Shopify" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Read test-order.json to get SKUs
$testOrderPath = "test-order.json"
if (-not (Test-Path $testOrderPath)) {
    Write-Host "❌ test-order.json not found!" -ForegroundColor Red
    exit 1
}

$testOrder = Get-Content $testOrderPath -Raw | ConvertFrom-Json
$skusToMap = $testOrder.items | ForEach-Object { $_.sku }

Write-Host "SKUs to map from test-order.json:" -ForegroundColor Yellow
foreach ($sku in $skusToMap) {
    Write-Host "  - $sku" -ForegroundColor Gray
}
Write-Host ""

# Step 1: List all SKUs from Shopify
Write-Host "[1/3] Fetching SKUs from Shopify..." -ForegroundColor Yellow
$shopifySkusOutput = & go run cmd/list-skus/main.go 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to fetch SKUs from Shopify" -ForegroundColor Red
    Write-Host $shopifySkusOutput -ForegroundColor Red
    exit 1
}

Write-Host "✅ Fetched SKUs from Shopify" -ForegroundColor Green
Write-Host ""

# Step 2: Find each SKU in Shopify
Write-Host "[2/3] Searching for SKUs in Shopify..." -ForegroundColor Yellow
$mappings = @()

foreach ($sku in $skusToMap) {
    Write-Host "  Searching for: $sku" -ForegroundColor Gray
    
    # Use find-sku command to search
    $findOutput = & go run cmd/find-sku/main.go $sku 2>&1 | Out-String
    
    if ($LASTEXITCODE -eq 0) {
        # Parse the output to extract IDs
        # The output format from find-sku is:
        # Product ID: 123
        # Variant ID: 456
        
        $productIDMatch = [regex]::Match($findOutput, "Product ID:\s*(\d+)")
        $variantIDMatch = [regex]::Match($findOutput, "Variant ID:\s*(\d+)")
        
        if ($productIDMatch.Success -and $variantIDMatch.Success) {
            $productID = $productIDMatch.Groups[1].Value
            $variantID = $variantIDMatch.Groups[1].Value
            
            Write-Host "    ✅ Found in Shopify" -ForegroundColor Green
            Write-Host "      Product ID: $productID, Variant ID: $variantID" -ForegroundColor Gray
            
            $mappings += @{
                SKU = $sku
                ProductID = $productID
                VariantID = $variantID
                Found = $true
            }
        } else {
            Write-Host "    ⚠️  Found but couldn't parse IDs" -ForegroundColor Yellow
            Write-Host "    Output: $findOutput" -ForegroundColor Gray
            $mappings += @{
                SKU = $sku
                Found = $false
            }
        }
    } else {
        Write-Host "    ❌ Not found in Shopify (will be treated as custom product)" -ForegroundColor Yellow
        $mappings += @{
            SKU = $sku
            Found = $false
        }
    }
}

Write-Host ""

# Step 3: Create mappings
Write-Host "[3/3] Creating SKU mappings..." -ForegroundColor Yellow
$mappedCount = 0

foreach ($mapping in $mappings) {
    if ($mapping.Found) {
        Write-Host "  Mapping SKU: $($mapping.SKU)" -ForegroundColor Gray
        
        $addOutput = & go run cmd/add-sku/main.go $mapping.SKU $mapping.ProductID $mapping.VariantID 2>&1
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "    ✅ Mapped successfully" -ForegroundColor Green
            $mappedCount++
        } else {
            Write-Host "    ❌ Failed to create mapping" -ForegroundColor Red
            Write-Host "    Error: $addOutput" -ForegroundColor Red
        }
    } else {
        Write-Host "  Skipping SKU: $($mapping.SKU) (not found in Shopify - will be custom product)" -ForegroundColor Gray
    }
}

Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Mapping Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "✅ Successfully mapped: $mappedCount out of $($skusToMap.Count) SKUs" -ForegroundColor Green
Write-Host ""

# Show unmapped SKUs
$unmapped = $mappings | Where-Object { -not $_.Found }
if ($unmapped.Count -gt 0) {
    Write-Host "⚠️  Unmapped SKUs (will be treated as custom products):" -ForegroundColor Yellow
    foreach ($item in $unmapped) {
        Write-Host "  - $($item.SKU)" -ForegroundColor Gray
    }
    Write-Host ""
    Write-Host "Note: Unmapped SKUs will still work - they'll be added as custom line items" -ForegroundColor Gray
    Write-Host "      in Shopify draft orders with the product_url provided." -ForegroundColor Gray
}

Write-Host ""
Write-Host "Next step: Run .\test-order.ps1 to test order submission" -ForegroundColor Cyan
Write-Host ""

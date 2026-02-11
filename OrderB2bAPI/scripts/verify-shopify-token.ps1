# Verify SHOPIFY_ACCESS_TOKEN works against Shopify Admin API
# Run from b2b/ or set env: SHOPIFY_SHOP_DOMAIN, SHOPIFY_ACCESS_TOKEN
# Usage: .\scripts\verify-shopify-token.ps1

# .env is in repo root (b2b/), one level up from OrderB2bAPI
$rootDir = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
$envPath = Join-Path $rootDir ".env"
if (Test-Path $envPath) {
    Write-Host "  .env: $envPath" -ForegroundColor Gray
    Get-Content $envPath | ForEach-Object {
        if ($_ -match '^\s*SHOPIFY_SHOP_DOMAIN=(.+)$') { $script:Shop = $matches[1].Trim() }
        if ($_ -match '^\s*SHOPIFY_ACCESS_TOKEN=(.+)$') { $script:Token = $matches[1].Trim() }
    }
} else {
    Write-Host "  .env not found at: $envPath" -ForegroundColor Yellow
}
$Shop = if ($env:SHOPIFY_SHOP_DOMAIN) { $env:SHOPIFY_SHOP_DOMAIN } else { $Shop }
$Token = if ($env:SHOPIFY_ACCESS_TOKEN) { $env:SHOPIFY_ACCESS_TOKEN } else { $Token }

if (-not $Shop -or -not $Token) {
    Write-Host "Set SHOPIFY_SHOP_DOMAIN and SHOPIFY_ACCESS_TOKEN (in .env or env vars)" -ForegroundColor Red
    exit 1
}

$Shop = $Shop -replace '^https?://', '' -replace '/$', ''
$url = "https://${Shop}/admin/api/2026-01/graphql.json"
$body = '{"query":"{ shop { name } }"}'

Write-Host "Testing Shopify token..." -ForegroundColor Cyan
Write-Host "  Shop: $Shop" -ForegroundColor Gray
Write-Host "  Token: $($Token.Substring(0, [Math]::Min(15, $Token.Length)))..." -ForegroundColor Gray

try {
    $r = Invoke-RestMethod -Uri $url -Method POST -Headers @{
        "Content-Type" = "application/json"
        "X-Shopify-Access-Token" = $Token
    } -Body $body
    Write-Host "OK - Token is valid. Shop name: $($r.data.shop.name)" -ForegroundColor Green
} catch {
    $code = $_.Exception.Response.StatusCode.value__
    Write-Host "FAIL - Shopify returned $code" -ForegroundColor Red
    if ($_.ErrorDetails.Message) { Write-Host $_.ErrorDetails.Message -ForegroundColor Red }
    Write-Host "Fix: In Shopify Admin, check your app's API token has 'write_draft_orders' (and read_customers/write_customers if needed). Regenerate the token if expired." -ForegroundColor Yellow
    exit 1
}

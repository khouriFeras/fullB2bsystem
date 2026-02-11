# Test script for GET /debug/menu - fetches a menu and the item with smallest branch
$baseUrl = "http://localhost:3000"

Write-Host "Testing /debug/menu (first menu, no query)..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/debug/menu" -UseBasicParsing
    $json = $response.Content | ConvertFrom-Json
    Write-Host "  Menu: $($json.menu.title) (handle: $($json.menu.handle), id: $($json.menu.id))" -ForegroundColor Green
    if ($json.smallestBranchItem) {
        Write-Host "  Smallest branch item: $($json.smallestBranchItem.title)" -ForegroundColor Green
        Write-Host "    id: $($json.smallestBranchItem.id), branchSize: $($json.smallestBranchItem.branchSize), type: $($json.smallestBranchItem.type)" -ForegroundColor Gray
        if ($json.smallestBranchItem.url) { Write-Host "    url: $($json.smallestBranchItem.url)" -ForegroundColor Gray }
    } else {
        Write-Host "  (No items in menu)" -ForegroundColor Yellow
    }
    Write-Host ""
    Write-Host "Full JSON saved to menu-response.json (UTF-8)" -ForegroundColor Gray
    # Write UTF-8 no BOM so Arabic and other Unicode display correctly in editors
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText("$PWD\menu-response.json", $response.Content, $utf8NoBom)
} catch {
    Write-Host "  Error: $_" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $reader.BaseStream.Position = 0
        Write-Host $reader.ReadToEnd() -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Optional: test with a specific menu handle, e.g.:" -ForegroundColor Cyan
Write-Host "  Invoke-WebRequest -Uri '$baseUrl/debug/menu?handle=main-menu' -UseBasicParsing" -ForegroundColor Gray

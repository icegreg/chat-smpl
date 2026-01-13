# Conference Chat Panel Test Runner
# Runs the conference chat panel E2E test with WebRTC support

$ErrorActionPreference = "Continue"

# Set environment for non-headless mode (required for WebRTC)
$env:HEADLESS = "false"

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "Conference Chat Panel E2E Test" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "This test requires:" -ForegroundColor Yellow
Write-Host "  - All services running (docker-compose up)" -ForegroundColor Yellow
Write-Host "  - FreeSWITCH with Verto WebSocket" -ForegroundColor Yellow
Write-Host "  - Chrome browser installed" -ForegroundColor Yellow
Write-Host ""

# Change to e2e-selenium directory
Push-Location $PSScriptRoot\e2e-selenium

try {
    Write-Host "Building TypeScript..." -ForegroundColor Green
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "TypeScript build failed" -ForegroundColor Red
        exit 1
    }

    Write-Host ""
    Write-Host "Running conference chat panel test..." -ForegroundColor Green
    Write-Host ""

    # Run the specific test file
    npx mocha dist/tests/conference-chat-panel.spec.js --timeout 180000

    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "============================================" -ForegroundColor Green
        Write-Host "All tests passed!" -ForegroundColor Green
        Write-Host "============================================" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "============================================" -ForegroundColor Red
        Write-Host "Some tests failed" -ForegroundColor Red
        Write-Host "============================================" -ForegroundColor Red
    }
} finally {
    Pop-Location
}

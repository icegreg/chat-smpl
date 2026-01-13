$ErrorActionPreference = "Stop"

Write-Host "Starting File Group Access E2E test..." -ForegroundColor Cyan
Write-Host ""

# Navigate to the correct directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

# Check if node_modules exists
if (-not (Test-Path "node_modules")) {
    Write-Host "Installing dependencies..." -ForegroundColor Yellow
    npm install
}

# Set environment variables
$env:HEADLESS = if ($env:HEADLESS) { $env:HEADLESS } else { "true" }
$env:BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://127.0.0.1:8888" }

Write-Host "Running with:" -ForegroundColor Yellow
Write-Host "  HEADLESS=$($env:HEADLESS)"
Write-Host "  BASE_URL=$($env:BASE_URL)"
Write-Host ""

# Run the test
npx mocha --import=tsx e2e-selenium/tests/file-group-access.spec.ts --timeout 180000 --exit

Write-Host ""
Write-Host "Test completed!" -ForegroundColor Green

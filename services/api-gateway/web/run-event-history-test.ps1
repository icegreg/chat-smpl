# Run Event History E2E tests
# Tests the event history panel functionality for regular users and moderators

$ErrorActionPreference = "Continue"

Write-Host "=== Running Event History Tests ===" -ForegroundColor Cyan

# Change to web directory
Set-Location $PSScriptRoot

# Check if node_modules exists
if (-not (Test-Path "e2e-selenium/node_modules")) {
    Write-Host "Installing dependencies..." -ForegroundColor Yellow
    Set-Location e2e-selenium
    npm install
    Set-Location ..
}

# Set environment variables
$env:BASE_URL = "http://localhost:8888"
$env:HEADLESS = "true"

# Run the tests
Write-Host "`nStarting Event History tests..." -ForegroundColor Green
Write-Host "BASE_URL: $env:BASE_URL" -ForegroundColor Gray

Set-Location e2e-selenium
npx mocha --require ts-node/register --timeout 120000 tests/event-history.spec.ts

$exitCode = $LASTEXITCODE
Set-Location ..

if ($exitCode -eq 0) {
    Write-Host "`n=== Event History Tests PASSED ===" -ForegroundColor Green
} else {
    Write-Host "`n=== Event History Tests FAILED ===" -ForegroundColor Red
}

exit $exitCode

# Run Active Events Sidebar E2E Test
# This test verifies the "Активные мероприятия" section in ChatSidebar

param(
    [switch]$Headless = $false
)

$ErrorActionPreference = "Stop"

Write-Host "=== Active Events Sidebar E2E Test ===" -ForegroundColor Cyan

# Set environment variables
$env:HEADLESS = if ($Headless) { "true" } else { "false" }
$env:TEST_URL = "http://localhost:8888"

Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  HEADLESS: $env:HEADLESS"
Write-Host "  TEST_URL: $env:TEST_URL"

# Change to web directory
$webDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $webDir

# Check if node_modules exists
if (-not (Test-Path "node_modules")) {
    Write-Host "Installing dependencies..." -ForegroundColor Yellow
    npm install
}

# Check if e2e-selenium/node_modules exists
if (-not (Test-Path "e2e-selenium/node_modules")) {
    Write-Host "Installing e2e-selenium dependencies..." -ForegroundColor Yellow
    Push-Location "e2e-selenium"
    npm install
    Pop-Location
}

# Compile TypeScript
Write-Host "`nCompiling TypeScript..." -ForegroundColor Yellow
Push-Location "e2e-selenium"
npx tsc --skipLibCheck
if ($LASTEXITCODE -ne 0) {
    Write-Host "TypeScript compilation failed!" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location

# Run the test
Write-Host "`nRunning Active Events Sidebar tests..." -ForegroundColor Yellow
Push-Location "e2e-selenium"

npx mocha dist/tests/active-events-sidebar.spec.js `
    --timeout 180000 `
    --slow 30000 `
    --reporter spec

$testResult = $LASTEXITCODE
Pop-Location

if ($testResult -eq 0) {
    Write-Host "`n=== All tests passed! ===" -ForegroundColor Green
} else {
    Write-Host "`n=== Some tests failed ===" -ForegroundColor Red
}

exit $testResult

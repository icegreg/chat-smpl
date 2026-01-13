# Run Voice Active Indicators E2E tests
# Tests: call indicator in chat list, Join button visibility

$ErrorActionPreference = "Continue"

Write-Host "=== Voice Active Indicators E2E Test ===" -ForegroundColor Cyan
Write-Host ""

# Set working directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

# Environment variables
$env:HEADLESS = "true"
$env:BASE_URL = "https://10.99.22.46"
$env:TS_NODE_TRANSPILE_ONLY = "true"
$env:NODE_TLS_REJECT_UNAUTHORIZED = "0"

Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  BASE_URL: $env:BASE_URL"
Write-Host "  HEADLESS: $env:HEADLESS"
Write-Host ""

# Run the test
Write-Host "Running voice-active-indicators.spec.ts..." -ForegroundColor Green
Write-Host ""

$nodePath = "C:\Program Files\nodejs\node.exe"
& $nodePath node_modules/mocha/bin/mocha `
    --require ts-node/register `
    --timeout 120000 `
    --reporter spec `
    "e2e-selenium/tests/voice-active-indicators.spec.ts"

$exitCode = $LASTEXITCODE

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "=== Tests PASSED ===" -ForegroundColor Green
} else {
    Write-Host "=== Tests FAILED (exit code: $exitCode) ===" -ForegroundColor Red
}

exit $exitCode

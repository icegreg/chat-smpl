# Run Participant Management E2E Tests
# Usage: .\run-participant-test.ps1

$ErrorActionPreference = "Continue"

Write-Host "=== Running Participant Management E2E Tests ===" -ForegroundColor Cyan

# Set environment variables
$env:BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://127.0.0.1:8888" }
$env:HEADLESS = if ($env:HEADLESS) { $env:HEADLESS } else { "true" }

Write-Host "BASE_URL: $env:BASE_URL"
Write-Host "HEADLESS: $env:HEADLESS"

# Change to web directory
Push-Location $PSScriptRoot

try {
    # Run the test
    Write-Host "`nRunning participant management tests..." -ForegroundColor Yellow
    npx mocha --require ts-node/register --timeout 120000 "e2e-selenium/tests/participant-management.spec.ts"

    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n=== Tests Passed ===" -ForegroundColor Green
    } else {
        Write-Host "`n=== Tests Failed ===" -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}

# Run Sidebar Events E2E Test
# Usage: .\run-sidebar-events-test.ps1

$ErrorActionPreference = "Continue"

# Set environment variables
$env:HEADLESS = "false"
$env:BASE_URL = "http://127.0.0.1:8888"
$env:PATH = "C:\Program Files\nodejs;$env:PATH"

Write-Host "=== Running Sidebar Events E2E Test ===" -ForegroundColor Cyan
Write-Host "HEADLESS: $env:HEADLESS"
Write-Host "BASE_URL: $env:BASE_URL"
Write-Host ""

# Change to web directory
Set-Location $PSScriptRoot

# Run the test
& "C:\Program Files\nodejs\npx.cmd" mocha --require ts-node/register e2e-selenium/tests/sidebar-events.spec.ts --timeout 120000

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan

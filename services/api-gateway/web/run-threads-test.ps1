# Run Threads E2E Test
$ErrorActionPreference = "Continue"

Write-Host "Running Threads E2E Test..." -ForegroundColor Cyan

# Add Node.js to PATH
$env:PATH = "C:\Program Files\nodejs;$env:PATH"

# Set environment variables
$env:HEADLESS = "false"
$env:BASE_URL = "http://127.0.0.1:8888"

# Change to web directory
Set-Location $PSScriptRoot

# Run the test using full path to npm
& "C:\Program Files\nodejs\npm.cmd" run test:e2e -- --grep "Threads and Subthreads"

Write-Host "Test completed!" -ForegroundColor Green

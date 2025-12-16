# Run API User Simulation Tests with timing measurements
# Usage: .\run-api-tests.ps1 [scenario|multiuser|all]

param(
    [string]$TestType = "all",
    [string]$ApiUrl = "http://127.0.0.1:8888"
)

$ErrorActionPreference = "Continue"
Set-Location $PSScriptRoot

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  API User Simulation Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "API URL: $ApiUrl" -ForegroundColor Yellow
Write-Host "Test Type: $TestType" -ForegroundColor Yellow
Write-Host ""

# Set environment variable
$env:API_URL = $ApiUrl

# Check if API is available
Write-Host "Checking API availability..." -ForegroundColor Gray
try {
    $response = Invoke-WebRequest -Uri "$ApiUrl/api/auth/me" -Method GET -TimeoutSec 5 -ErrorAction SilentlyContinue
} catch {
    if ($_.Exception.Response.StatusCode -eq 401) {
        Write-Host "API is available (got 401 - expected for unauthenticated request)" -ForegroundColor Green
    } else {
        Write-Host "Warning: API may not be available at $ApiUrl" -ForegroundColor Yellow
        Write-Host "Error: $_" -ForegroundColor Yellow
    }
}

Write-Host ""

# Define test files
$scenarioTests = "e2e-selenium/tests/api-user-scenarios.spec.ts"
$multiuserTests = "e2e-selenium/tests/api-multiuser-simulation.spec.ts"
$authPresenceTests = "e2e-selenium/tests/api-auth-presence.spec.ts"

# Build mocha command based on test type
$testFiles = @()
switch ($TestType.ToLower()) {
    "scenario" {
        $testFiles += $scenarioTests
        Write-Host "Running: User Scenario Tests" -ForegroundColor Cyan
    }
    "multiuser" {
        $testFiles += $multiuserTests
        Write-Host "Running: Multi-User Simulation Tests" -ForegroundColor Cyan
    }
    "auth" {
        $testFiles += $authPresenceTests
        Write-Host "Running: Auth & Presence Tests" -ForegroundColor Cyan
    }
    "all" {
        $testFiles += $authPresenceTests
        $testFiles += $scenarioTests
        $testFiles += $multiuserTests
        Write-Host "Running: All API Tests" -ForegroundColor Cyan
    }
    default {
        Write-Host "Unknown test type: $TestType" -ForegroundColor Red
        Write-Host "Available options: scenario, multiuser, auth, all" -ForegroundColor Yellow
        exit 1
    }
}

Write-Host ""
Write-Host "Test files:" -ForegroundColor Gray
$testFiles | ForEach-Object { Write-Host "  - $_" -ForegroundColor Gray }
Write-Host ""

# Run tests
$startTime = Get-Date
Write-Host "Starting tests at $startTime" -ForegroundColor Gray
Write-Host "-".PadRight(60, '-') -ForegroundColor Gray
Write-Host ""

$testFileArgs = $testFiles -join " "
$mochaCmd = "npx mocha --require ts-node/register --timeout 300000 $testFileArgs"

Write-Host "Command: $mochaCmd" -ForegroundColor DarkGray
Write-Host ""

Invoke-Expression $mochaCmd
$exitCode = $LASTEXITCODE

$endTime = Get-Date
$duration = $endTime - $startTime

Write-Host ""
Write-Host "-".PadRight(60, '-') -ForegroundColor Gray
Write-Host "Tests completed at $endTime" -ForegroundColor Gray
Write-Host "Total duration: $($duration.ToString('mm\:ss'))" -ForegroundColor Gray
Write-Host ""

if ($exitCode -eq 0) {
    Write-Host "ALL TESTS PASSED" -ForegroundColor Green
} else {
    Write-Host "SOME TESTS FAILED" -ForegroundColor Red
}

exit $exitCode

# Run Reply and Forward E2E tests
# Usage: .\run-reply-forward-test.ps1 [-Test <test-type>]
# Test types: all, reply, forward, modal, flow, indicator

param(
    [string]$Test = "all",
    [switch]$Headless = $false
)

$ErrorActionPreference = "Stop"

# Set environment
if ($Headless) {
    $env:HEADLESS = "true"
} else {
    $env:HEADLESS = "false"
}

$env:BASE_URL = "http://127.0.0.1:8888"

Write-Host "Running Reply and Forward E2E tests..." -ForegroundColor Cyan
Write-Host "Base URL: $env:BASE_URL" -ForegroundColor Gray
Write-Host "Headless: $env:HEADLESS" -ForegroundColor Gray

# Navigate to web directory
Push-Location $PSScriptRoot

try {
    switch ($Test.ToLower()) {
        "all" {
            Write-Host "`nRunning all reply-forward tests..." -ForegroundColor Yellow
            npx mocha --require ts-node/register "e2e-selenium/tests/reply-forward.spec.ts" --timeout 120000
        }
        "reply" {
            Write-Host "`nRunning Reply Quote Display tests..." -ForegroundColor Yellow
            npx mocha --require ts-node/register "e2e-selenium/tests/reply-forward.spec.ts" --grep "Reply Quote Display" --timeout 120000
        }
        "forward" {
            Write-Host "`nRunning Forward Button tests..." -ForegroundColor Yellow
            npx mocha --require ts-node/register "e2e-selenium/tests/reply-forward.spec.ts" --grep "Forward Button" --timeout 120000
        }
        "modal" {
            Write-Host "`nRunning Forward Modal tests..." -ForegroundColor Yellow
            npx mocha --require ts-node/register "e2e-selenium/tests/reply-forward.spec.ts" --grep "Forward Modal" --timeout 120000
        }
        "flow" {
            Write-Host "`nRunning Forward Message Flow tests..." -ForegroundColor Yellow
            npx mocha --require ts-node/register "e2e-selenium/tests/reply-forward.spec.ts" --grep "Forward Message Flow" --timeout 120000
        }
        "indicator" {
            Write-Host "`nRunning Forwarded Message Indicator tests..." -ForegroundColor Yellow
            npx mocha --require ts-node/register "e2e-selenium/tests/reply-forward.spec.ts" --grep "Forwarded Message Indicator" --timeout 120000
        }
        default {
            Write-Host "Unknown test type: $Test" -ForegroundColor Red
            Write-Host "Available types: all, reply, forward, modal, flow, indicator" -ForegroundColor Yellow
            exit 1
        }
    }

    if ($LASTEXITCODE -eq 0) {
        Write-Host "`nTests passed!" -ForegroundColor Green
    } else {
        Write-Host "`nTests failed with exit code: $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
} finally {
    Pop-Location
}

# Run WebSocket Recovery E2E Tests
# Tests automatic message recovery after WebSocket disconnection

param(
    [switch]$Headless = $false,
    [string]$Test = "all"  # all, automatic, fallback, multidevice, seqnum, pending
)

$ErrorActionPreference = "Stop"
$env:PATH = "C:\Program Files\nodejs;" + $env:PATH

Set-Location $PSScriptRoot

Write-Host ""
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host "  WebSocket Recovery E2E Tests" -ForegroundColor Cyan
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host ""

# Set environment
$env:BASE_URL = "http://127.0.0.1:8888"
if ($Headless) {
    $env:HEADLESS = "true"
    Write-Host "Running in HEADLESS mode" -ForegroundColor Yellow
} else {
    $env:HEADLESS = "false"
    Write-Host "Running in VISIBLE mode (use -Headless for headless)" -ForegroundColor Gray
}

Write-Host "Base URL: $env:BASE_URL" -ForegroundColor Gray
Write-Host ""

# Determine which tests to run
$grepPattern = switch ($Test) {
    "automatic" { "should automatically receive missed messages" }
    "fallback" { "API Fallback" }
    "multidevice" { "Multi-Device" }
    "seqnum" { "seq_num Tracking" }
    "pending" { "Pending Messages" }
    default { "" }
}

Write-Host "Running tests: $Test" -ForegroundColor Cyan
Write-Host ""

# Build mocha arguments
$mochaArgs = @(
    "--loader=ts-node/esm",
    "--no-warnings",
    "./node_modules/mocha/bin/mocha.js",
    "--timeout", "300000",
    "--no-config",
    "--extension=ts"
)

if ($grepPattern) {
    $mochaArgs += "--grep"
    $mochaArgs += "`"$grepPattern`""
}

$mochaArgs += "e2e-selenium/tests/websocket-recovery.spec.ts"

# Run tests using node with ts-node/esm loader
& "C:\Program Files\nodejs\node.exe" $mochaArgs

$exitCode = $LASTEXITCODE

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "=" * 60 -ForegroundColor Green
    Write-Host "  ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "=" * 60 -ForegroundColor Green
} else {
    Write-Host "=" * 60 -ForegroundColor Red
    Write-Host "  SOME TESTS FAILED (exit code: $exitCode)" -ForegroundColor Red
    Write-Host "=" * 60 -ForegroundColor Red
}

exit $exitCode

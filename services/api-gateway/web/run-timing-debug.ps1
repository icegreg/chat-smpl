# Run voice timing debug test

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

$env:HEADLESS = "false"
$env:BASE_URL = "https://localhost"
$env:TS_NODE_TRANSPILE_ONLY = "true"
$env:NODE_TLS_REJECT_UNAUTHORIZED = "0"

Write-Host "Running Voice Timing Debug Test" -ForegroundColor Cyan
Write-Host "BASE_URL: $env:BASE_URL" -ForegroundColor Gray
Write-Host "HEADLESS: $env:HEADLESS" -ForegroundColor Gray
Write-Host ""

& "C:\Program Files\nodejs\node.exe" node_modules/mocha/bin/mocha `
    --require ts-node/register `
    --timeout 120000 `
    "e2e-selenium/tests/voice-timing-debug.spec.ts"

$exitCode = $LASTEXITCODE
Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "TEST PASSED" -ForegroundColor Green
} else {
    Write-Host "TEST FAILED (exit code: $exitCode)" -ForegroundColor Red
}

exit $exitCode

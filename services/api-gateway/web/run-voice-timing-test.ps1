# Run voice invite timing test (3 clients)
# This test verifies that all participants receive invites within 2 seconds

# Change to the script's directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptDir

$env:HEADLESS = "false"
$env:BASE_URL = "https://localhost"
$env:TS_NODE_TRANSPILE_ONLY = "true"
$env:NODE_TLS_REJECT_UNAUTHORIZED = "0"

Write-Host "Running Voice Invite Timing Test (3 clients)" -ForegroundColor Cyan
Write-Host "BASE_URL: $env:BASE_URL" -ForegroundColor Gray
Write-Host "HEADLESS: $env:HEADLESS" -ForegroundColor Gray
Write-Host "Working dir: $(Get-Location)" -ForegroundColor Gray
Write-Host ""

# Run the test
& "C:\Program Files\nodejs\node.exe" node_modules/mocha/bin/mocha `
    --require ts-node/register `
    --timeout 180000 `
    "e2e-selenium/tests/voice-invite-timing.spec.ts"

$exitCode = $LASTEXITCODE
Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "TEST PASSED" -ForegroundColor Green
} else {
    Write-Host "TEST FAILED (exit code: $exitCode)" -ForegroundColor Red
}

exit $exitCode

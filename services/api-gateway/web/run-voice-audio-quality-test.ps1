# Voice Audio Quality E2E Test Runner
# Tests actual audio signal transmission and quality
# Verifies: signal detection, frequency preservation, packet loss, jitter

$ErrorActionPreference = "Stop"
$env:PATH = "C:\Program Files\nodejs;" + $env:PATH

# Set working directory
Set-Location $PSScriptRoot

Write-Host "=== Voice Audio Quality E2E Test ===" -ForegroundColor Cyan
Write-Host ""

# Check Chrome version
try {
    $chromeVersion = (Get-Item "C:\Program Files\Google\Chrome\Application\chrome.exe").VersionInfo.FileVersion
    Write-Host "Chrome version: $chromeVersion" -ForegroundColor Gray
} catch {
    Write-Host "Warning: Could not detect Chrome version" -ForegroundColor Yellow
}

# Check FreeSWITCH status
Write-Host ""
Write-Host "Checking FreeSWITCH status..." -ForegroundColor Gray
try {
    $fsStatus = docker ps --filter "name=chatapp-freeswitch" --format "{{.Status}}"
    if ($fsStatus) {
        Write-Host "FreeSWITCH: $fsStatus" -ForegroundColor Green
    } else {
        Write-Host "FreeSWITCH: Not running!" -ForegroundColor Red
        Write-Host "Run 'docker-compose up -d freeswitch' to start FreeSWITCH" -ForegroundColor Yellow
    }
} catch {
    Write-Host "Could not check FreeSWITCH status" -ForegroundColor Yellow
}

# Check voice-service status
Write-Host ""
Write-Host "Checking voice-service status..." -ForegroundColor Gray
try {
    $voiceStatus = docker ps --filter "name=chatapp-voice" --format "{{.Status}}"
    if ($voiceStatus) {
        Write-Host "voice-service: $voiceStatus" -ForegroundColor Green
    } else {
        Write-Host "voice-service: Not running!" -ForegroundColor Red
    }
} catch {
    Write-Host "Could not check voice-service status" -ForegroundColor Yellow
}

# Set environment variables
$env:HEADLESS = "false"  # WebRTC works better with visible browser
$env:BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://127.0.0.1:8888" }

Write-Host ""
Write-Host "Test Configuration:" -ForegroundColor Cyan
Write-Host "  BASE_URL: $env:BASE_URL"
Write-Host "  HEADLESS: $env:HEADLESS"
Write-Host ""

Write-Host "This test verifies:" -ForegroundColor Cyan
Write-Host "  - Audio signal reaches receiver" -ForegroundColor White
Write-Host "  - Dominant frequency is preserved (~440Hz)" -ForegroundColor White
Write-Host "  - Packet loss < 5%" -ForegroundColor White
Write-Host "  - Jitter < 100ms" -ForegroundColor White
Write-Host "  - Quality score > 50/100" -ForegroundColor White
Write-Host ""

# Run the test
Write-Host "Running voice audio quality test..." -ForegroundColor Cyan
Write-Host "===========================================" -ForegroundColor Cyan
Write-Host ""

try {
    # Build mocha arguments - same pattern as other working tests
    $mochaArgs = @(
        "--loader=ts-node/esm",
        "--no-warnings",
        "./node_modules/mocha/bin/mocha.js",
        "--timeout", "300000",
        "--slow", "30000",
        "--no-config",
        "--extension=ts",
        "e2e-selenium/tests/voice-audio-quality.spec.ts"
    )

    # Run tests using node with ts-node/esm loader
    & "C:\Program Files\nodejs\node.exe" $mochaArgs

    $testResult = $LASTEXITCODE
} catch {
    Write-Host "Test execution error: $_" -ForegroundColor Red
    $testResult = 1
}

Write-Host ""
Write-Host "===========================================" -ForegroundColor Cyan

if ($testResult -eq 0) {
    Write-Host "Voice audio quality test PASSED" -ForegroundColor Green
} else {
    Write-Host "Voice audio quality test FAILED" -ForegroundColor Red
}

exit $testResult

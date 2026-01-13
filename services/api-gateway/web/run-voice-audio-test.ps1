# Voice Audio Transmission E2E Test Runner
# This test requires:
# - FreeSWITCH running with Verto WebSocket
# - Non-headless browser (HEADLESS=false by default)
# - Chrome with fake media device

$ErrorActionPreference = "Stop"

# Set working directory
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $scriptPath

Write-Host "=== Voice Audio Transmission E2E Test ===" -ForegroundColor Cyan
Write-Host ""

# Check if ChromeDriver is available
try {
    $chromeVersion = (Get-Item "C:\Program Files\Google\Chrome\Application\chrome.exe").VersionInfo.FileVersion
    Write-Host "Chrome version: $chromeVersion" -ForegroundColor Gray
} catch {
    Write-Host "Warning: Could not detect Chrome version" -ForegroundColor Yellow
}

# Check if FreeSWITCH is running
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

# Check voice-service
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

Write-Host ""
Write-Host "Building TypeScript..." -ForegroundColor Gray

# Build TypeScript
npm run build 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "TypeScript build failed" -ForegroundColor Red
    exit 1
}
Write-Host "TypeScript build complete" -ForegroundColor Green

# Set environment variables
$env:HEADLESS = "false"  # WebRTC works better with visible browser
$env:BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://127.0.0.1:8888" }

Write-Host ""
Write-Host "Test Configuration:" -ForegroundColor Cyan
Write-Host "  BASE_URL: $env:BASE_URL"
Write-Host "  HEADLESS: $env:HEADLESS"
Write-Host ""

# Run the test
Write-Host "Running voice audio transmission test..." -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

try {
    # Run with increased timeout and verbose output
    npx mocha `
        --require ts-node/register `
        --timeout 180000 `
        --slow 30000 `
        --reporter spec `
        "dist/tests/voice-audio-transmission.spec.js"

    $testResult = $LASTEXITCODE
} catch {
    Write-Host "Test execution error: $_" -ForegroundColor Red
    $testResult = 1
}

Write-Host ""
Write-Host "==========================================" -ForegroundColor Cyan

if ($testResult -eq 0) {
    Write-Host "Voice audio transmission test PASSED" -ForegroundColor Green
} else {
    Write-Host "Voice audio transmission test FAILED" -ForegroundColor Red
}

exit $testResult

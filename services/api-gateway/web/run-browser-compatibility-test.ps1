# Voice Browser Compatibility Test Runner
# Tests conference creation in Chrome and Firefox with both localhost and real IP

$ErrorActionPreference = "Stop"

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Voice Browser Compatibility Tests" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "This test will verify:" -ForegroundColor Yellow
Write-Host "  1. Conference creation in Chrome and Firefox" -ForegroundColor Yellow
Write-Host "  2. ConferenceView popup appears" -ForegroundColor Yellow
Write-Host "  3. Works with localhost and real IP (192.168.1.208)" -ForegroundColor Yellow
Write-Host "  4. No SDP parsing errors (auto-nat fix verification)" -ForegroundColor Yellow
Write-Host ""

# Check if we're in the right directory
if (-not (Test-Path "e2e-selenium")) {
    Write-Host "Error: Please run this script from services/api-gateway/web directory" -ForegroundColor Red
    exit 1
}

# Check if frontend is running
Write-Host "Checking if frontend is running..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8888" -Method Head -TimeoutSec 5 -ErrorAction Stop
    Write-Host "✓ Frontend is running on localhost:8888" -ForegroundColor Green
} catch {
    Write-Host "✗ Frontend is not running on localhost:8888" -ForegroundColor Red
    Write-Host "  Please start the frontend first: npm run dev" -ForegroundColor Yellow
    exit 1
}

# Check if FreeSWITCH is running
Write-Host "Checking if FreeSWITCH is running..." -ForegroundColor Yellow
try {
    $fsStatus = docker-compose ps freeswitch 2>&1
    if ($fsStatus -match "running" -or $fsStatus -match "Up") {
        Write-Host "✓ FreeSWITCH is running" -ForegroundColor Green
    } else {
        Write-Host "✗ FreeSWITCH is not running" -ForegroundColor Red
        Write-Host "  Please start FreeSWITCH first: docker-compose up -d freeswitch" -ForegroundColor Yellow
        exit 1
    }
} catch {
    Write-Host "✗ Could not check FreeSWITCH status" -ForegroundColor Red
    exit 1
}

# Check EXTERNAL_RTP_IP setting
Write-Host "Checking FreeSWITCH EXTERNAL_RTP_IP..." -ForegroundColor Yellow
$externalRtpIp = docker-compose exec -T freeswitch sh -c 'echo $EXTERNAL_RTP_IP' 2>$null
if ($externalRtpIp) {
    $externalRtpIp = $externalRtpIp.Trim()
    Write-Host "✓ EXTERNAL_RTP_IP is set to: $externalRtpIp" -ForegroundColor Green

    if ($externalRtpIp -eq "auto-nat") {
        Write-Host "  WARNING: EXTERNAL_RTP_IP is still auto-nat - Firefox may fail!" -ForegroundColor Yellow
        Write-Host "  Expected: real IP (e.g., 192.168.1.208) or auto" -ForegroundColor Yellow
    }
} else {
    Write-Host "✗ Could not read EXTERNAL_RTP_IP" -ForegroundColor Red
}

Write-Host ""
Write-Host "Starting tests..." -ForegroundColor Cyan
Write-Host ""

# Set environment variables
$env:HEADLESS = "false"
$env:BASE_URL = "http://localhost:8888"

# Run the test
Write-Host "Running mocha tests..." -ForegroundColor Yellow
Write-Host ""

npm run test:e2e:headed -- --spec "e2e-selenium/tests/voice-browser-compatibility.spec.ts"

$exitCode = $LASTEXITCODE

Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "==================================" -ForegroundColor Green
    Write-Host "All tests passed successfully! ✓" -ForegroundColor Green
    Write-Host "==================================" -ForegroundColor Green
} else {
    Write-Host "==================================" -ForegroundColor Red
    Write-Host "Some tests failed! ✗" -ForegroundColor Red
    Write-Host "==================================" -ForegroundColor Red
}

exit $exitCode

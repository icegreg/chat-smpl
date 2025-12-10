# Extreme Load Test Orchestrator
# Запуск полного нагрузочного теста:
# - 400 k6 VU генерируют 10 msg/sec
# - 10 браузеров с медленной сетью
# - 20 браузеров с обрывающейся сетью
# - 20% сообщений с файлами (UUID внутри)

param(
    [int]$K6Users = 400,
    [int]$TargetMPS = 10,
    [int]$SlowBrowsers = 10,
    [int]$FlakyBrowsers = 20,
    [int]$DurationSeconds = 300,
    [double]$FileRatio = 0.2,
    [string]$BaseUrl = "http://127.0.0.1:8888",
    [switch]$K6Only,
    [switch]$BrowsersOnly,
    [switch]$DryRun
)

$ErrorActionPreference = "Continue"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$webDir = "$scriptDir\..\services\api-gateway\web"

# Add Node.js to PATH
$nodePath = "C:\Program Files\nodejs"
if (Test-Path $nodePath) {
    $env:PATH = "$nodePath;$env:PATH"
}

Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  EXTREME LOAD TEST" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""
Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  Base URL:        $BaseUrl"
Write-Host "  k6 Users:        $K6Users"
Write-Host "  Target MPS:      $TargetMPS msg/sec"
Write-Host "  Slow Browsers:   $SlowBrowsers"
Write-Host "  Flaky Browsers:  $FlakyBrowsers"
Write-Host "  Duration:        $DurationSeconds seconds"
Write-Host "  File Ratio:      $($FileRatio * 100)%"
Write-Host ""

if ($DryRun) {
    Write-Host "DRY RUN - Not actually starting tests" -ForegroundColor Yellow
    exit 0
}

# Check server availability
Write-Host "Checking server availability..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri $BaseUrl -TimeoutSec 10 -UseBasicParsing
    Write-Host "  Server is available (HTTP $($response.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "  ERROR: Server not available at $BaseUrl" -ForegroundColor Red
    Write-Host "  Make sure docker compose up is running" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Step 1: Create shared chat and get tokens
Write-Host "Step 1: Creating shared chat..." -ForegroundColor Cyan

$ownerUsername = "extreme_owner_$(Get-Random -Maximum 99999)"
$ownerEmail = "$ownerUsername@extreme.local"

# Register owner
$registerBody = @{
    username = $ownerUsername
    email = $ownerEmail
    password = "TestPass123!"
} | ConvertTo-Json

try {
    $registerRes = Invoke-RestMethod -Uri "$BaseUrl/api/auth/register" `
        -Method POST `
        -ContentType "application/json" `
        -Body $registerBody

    $ownerToken = $registerRes.access_token
    Write-Host "  Owner registered: $ownerUsername" -ForegroundColor Green
} catch {
    Write-Host "  Failed to register owner: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Get owner ID
try {
    $meRes = Invoke-RestMethod -Uri "$BaseUrl/api/auth/me" `
        -Method GET `
        -Headers @{ Authorization = "Bearer $ownerToken" }
    $ownerId = $meRes.id
} catch {
    Write-Host "  Failed to get owner info" -ForegroundColor Red
    exit 1
}

# Create chat
$chatBody = @{
    type = "group"
    name = "Extreme Load Test $(Get-Date -Format 'yyyy-MM-dd HH:mm')"
    description = "$K6Users k6 users + $SlowBrowsers slow browsers + $FlakyBrowsers flaky browsers"
    participant_ids = @()
} | ConvertTo-Json

try {
    $chatRes = Invoke-RestMethod -Uri "$BaseUrl/api/chats" `
        -Method POST `
        -ContentType "application/json" `
        -Headers @{ Authorization = "Bearer $ownerToken" } `
        -Body $chatBody

    $chatId = $chatRes.id
    Write-Host "  Chat created: $chatId" -ForegroundColor Green
} catch {
    Write-Host "  Failed to create chat: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Export for child processes
$env:CHAT_ID = $chatId
$env:OWNER_TOKEN = $ownerToken
$env:BASE_URL = $BaseUrl

# Step 2: Start k6 load generator
$k6Job = $null
if (-not $BrowsersOnly) {
    Write-Host "Step 2: Starting k6 load generator ($K6Users VUs, $TargetMPS msg/sec)..." -ForegroundColor Cyan

    $k6Script = "$scriptDir\extreme-load-test.js"

    # Check if k6 is installed locally
    $k6Installed = Get-Command k6 -ErrorAction SilentlyContinue

    if ($k6Installed) {
        Write-Host "  Using local k6" -ForegroundColor Green

        $k6Job = Start-Job -ScriptBlock {
            param($scriptDir, $k6Script, $BaseUrl, $K6Users, $TargetMPS, $FileRatio, $DurationSeconds)

            Set-Location $scriptDir
            $env:BASE_URL = $BaseUrl
            $env:VUS = $K6Users
            $env:TARGET_MPS = $TargetMPS
            $env:FILE_RATIO = $FileRatio
            $env:DURATION = "${DurationSeconds}s"

            k6 run `
                --vus $K6Users `
                --duration "${DurationSeconds}s" `
                -e BASE_URL=$BaseUrl `
                -e VUS=$K6Users `
                -e TARGET_MPS=$TargetMPS `
                -e FILE_RATIO=$FileRatio `
                $k6Script 2>&1
        } -ArgumentList $scriptDir, $k6Script, $BaseUrl, $K6Users, $TargetMPS, $FileRatio, $DurationSeconds

        Write-Host "  k6 started (Job ID: $($k6Job.Id))" -ForegroundColor Green
    } else {
        Write-Host "  Using Docker k6" -ForegroundColor Green

        $k6Job = Start-Job -ScriptBlock {
            param($scriptDir, $BaseUrl, $K6Users, $TargetMPS, $FileRatio, $DurationSeconds)

            Set-Location $scriptDir
            # Using network_mode: host - k6 connects directly to localhost
            docker compose run --rm `
                -e BASE_URL=http://127.0.0.1:8888 `
                -e WS_URL=ws://127.0.0.1:8000 `
                -e VUS=$K6Users `
                -e TARGET_MPS=$TargetMPS `
                -e FILE_RATIO=$FileRatio `
                -e DURATION="${DurationSeconds}s" `
                k6 run `
                --vus $K6Users `
                --duration "${DurationSeconds}s" `
                extreme-load-test.js 2>&1
        } -ArgumentList $scriptDir, $BaseUrl, $K6Users, $TargetMPS, $FileRatio, $DurationSeconds

        Write-Host "  k6 Docker started (Job ID: $($k6Job.Id))" -ForegroundColor Green
    }
}
Write-Host ""

# Step 3: Start browser clients
$browserJob = $null
if (-not $K6Only) {
    Write-Host "Step 3: Starting browser clients ($SlowBrowsers slow + $FlakyBrowsers flaky)..." -ForegroundColor Cyan

    # Delay browser start to let k6 ramp up
    Write-Host "  Waiting 10 seconds for k6 to ramp up..." -ForegroundColor Yellow
    Start-Sleep -Seconds 10

    $browserJob = Start-Job -ScriptBlock {
        param($webDir, $chatId, $ownerToken, $BaseUrl, $SlowBrowsers, $FlakyBrowsers, $DurationSeconds, $nodePath)

        $env:PATH = "$nodePath;$env:PATH"
        $env:CHAT_ID = $chatId
        $env:OWNER_TOKEN = $ownerToken
        $env:BASE_URL = $BaseUrl
        $env:SLOW_CLIENTS = $SlowBrowsers
        $env:FLAKY_CLIENTS = $FlakyBrowsers
        $env:TEST_DURATION = $DurationSeconds
        $env:HEADLESS = "true"

        Set-Location $webDir
        npm run test:e2e -- --grep "Extreme Browser Clients" 2>&1
    } -ArgumentList $webDir, $chatId, $ownerToken, $BaseUrl, $SlowBrowsers, $FlakyBrowsers, $DurationSeconds, $nodePath

    Write-Host "  Browser tests started (Job ID: $($browserJob.Id))" -ForegroundColor Green
}
Write-Host ""

# Step 4: Monitor progress
Write-Host "Step 4: Monitoring test progress..." -ForegroundColor Cyan
Write-Host "  Press Ctrl+C to stop" -ForegroundColor Yellow
Write-Host ""

$startTime = Get-Date
$endTime = $startTime.AddSeconds($DurationSeconds + 60)  # Extra minute for cleanup

try {
    while ((Get-Date) -lt $endTime) {
        $elapsed = [int]((Get-Date) - $startTime).TotalSeconds
        $remaining = [int]($DurationSeconds - $elapsed)

        if ($remaining -lt 0) { $remaining = 0 }

        Write-Host "`r  Elapsed: ${elapsed}s | Remaining: ${remaining}s  " -NoNewline

        # Check job status
        if ($k6Job -and $k6Job.State -eq 'Completed') {
            Write-Host ""
            Write-Host "  k6 job completed" -ForegroundColor Green
            $k6Output = Receive-Job $k6Job
            $k6Job = $null
        }

        if ($browserJob -and $browserJob.State -eq 'Completed') {
            Write-Host ""
            Write-Host "  Browser job completed" -ForegroundColor Green
            $browserOutput = Receive-Job $browserJob
            $browserJob = $null
        }

        if (-not $k6Job -and -not $browserJob) {
            break
        }

        Start-Sleep -Seconds 5
    }
} finally {
    Write-Host ""
}

# Step 5: Collect results
Write-Host ""
Write-Host "Step 5: Collecting results..." -ForegroundColor Cyan

if ($k6Job) {
    Write-Host ""
    Write-Host "k6 Results:" -ForegroundColor Yellow
    $k6Output = Receive-Job $k6Job -Wait
    # Show last 50 lines of k6 output
    $k6Output | Select-Object -Last 50 | ForEach-Object { Write-Host "  $_" }
    Remove-Job $k6Job -Force
}

if ($browserJob) {
    Write-Host ""
    Write-Host "Browser Results:" -ForegroundColor Yellow
    $browserOutput = Receive-Job $browserJob -Wait
    # Show last 30 lines
    $browserOutput | Select-Object -Last 30 | ForEach-Object { Write-Host "  $_" }
    Remove-Job $browserJob -Force
}

# Summary
Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  TEST COMPLETED" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""
Write-Host "Chat ID: $chatId" -ForegroundColor Yellow
Write-Host "You can view the chat at: $BaseUrl/chat/$chatId" -ForegroundColor Yellow
Write-Host ""

# Get message count from API
try {
    $messagesRes = Invoke-RestMethod -Uri "$BaseUrl/api/chats/$chatId/messages?limit=1" `
        -Method GET `
        -Headers @{ Authorization = "Bearer $ownerToken" }

    Write-Host "Total messages in chat: $($messagesRes.total)" -ForegroundColor Green
} catch {
    Write-Host "Could not get message count" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Test completed!" -ForegroundColor Green

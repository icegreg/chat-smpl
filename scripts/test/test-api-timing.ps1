# Quick API Performance Test - Standalone PowerShell script
# Tests basic API operations with timing measurements
# Usage: .\test-api-timing.ps1 [-ApiUrl "http://127.0.0.1:8888"] [-Verbose]

param(
    [string]$ApiUrl = "http://127.0.0.1:8888",
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Timing thresholds (ms)
$EXCELLENT = 100
$GOOD = 300
$ACCEPTABLE = 1000

# Results storage
$results = @()

function Get-Rating {
    param([int]$Duration)
    if ($Duration -lt $EXCELLENT) { return "EXCELLENT" }
    if ($Duration -lt $GOOD) { return "GOOD" }
    if ($Duration -lt $ACCEPTABLE) { return "ACCEPTABLE" }
    return "SLOW"
}

function Get-RatingColor {
    param([string]$Rating)
    switch ($Rating) {
        "EXCELLENT" { return "Green" }
        "GOOD" { return "Cyan" }
        "ACCEPTABLE" { return "Yellow" }
        "SLOW" { return "Red" }
    }
}

function Invoke-TimedRequest {
    param(
        [string]$Method,
        [string]$Path,
        [hashtable]$Headers = @{},
        [string]$Body = $null,
        [string]$Description
    )

    $url = "$ApiUrl$Path"
    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

    try {
        $params = @{
            Uri = $url
            Method = $Method
            Headers = $Headers
            ContentType = "application/json"
        }
        if ($Body) {
            $params.Body = $Body
        }

        $response = Invoke-RestMethod @params
        $stopwatch.Stop()
        $duration = $stopwatch.ElapsedMilliseconds
        $rating = Get-Rating -Duration $duration
        $color = Get-RatingColor -Rating $rating

        $script:results += @{
            Description = $Description
            Method = $Method
            Path = $Path
            Duration = $duration
            Rating = $rating
            Success = $true
        }

        if ($Verbose) {
            Write-Host "  [$rating] " -NoNewline -ForegroundColor $color
            Write-Host "${duration}ms - $Description" -ForegroundColor Gray
        }

        return $response
    }
    catch {
        $stopwatch.Stop()
        $duration = $stopwatch.ElapsedMilliseconds

        $script:results += @{
            Description = $Description
            Method = $Method
            Path = $Path
            Duration = $duration
            Rating = "FAILED"
            Success = $false
            Error = $_.Exception.Message
        }

        if ($Verbose) {
            Write-Host "  [FAILED] " -NoNewline -ForegroundColor Red
            Write-Host "${duration}ms - $Description - $_" -ForegroundColor Gray
        }

        return $null
    }
}

# Header
Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  API PERFORMANCE TEST" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""
Write-Host "API URL: $ApiUrl" -ForegroundColor Yellow
Write-Host "Thresholds: EXCELLENT < ${EXCELLENT}ms, GOOD < ${GOOD}ms, ACCEPTABLE < ${ACCEPTABLE}ms" -ForegroundColor Gray
Write-Host ""

$testStartTime = Get-Date

# Generate unique test user
$timestamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
$random = -join ((97..122) | Get-Random -Count 6 | ForEach-Object { [char]$_ })
$username = "perftest_${timestamp}_${random}"
$email = "${username}@test.local"
$password = "TestPass123!"

Write-Host "Test user: $username" -ForegroundColor Gray
Write-Host ""

# ==================== TESTS ====================

Write-Host "1. Authentication Tests" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

# Register
$regBody = @{ username = $username; email = $email; password = $password } | ConvertTo-Json
$authResult = Invoke-TimedRequest -Method "POST" -Path "/api/auth/register" -Body $regBody -Description "Register new user"

if (-not $authResult) {
    Write-Host "Registration failed, cannot continue" -ForegroundColor Red
    exit 1
}

$token = $authResult.access_token
$refreshToken = $authResult.refresh_token
$authHeaders = @{ Authorization = "Bearer $token" }

# Get profile
Invoke-TimedRequest -Method "GET" -Path "/api/auth/me" -Headers $authHeaders -Description "Get user profile"

# Refresh token
$refreshBody = @{ refresh_token = $refreshToken } | ConvertTo-Json
$refreshResult = Invoke-TimedRequest -Method "POST" -Path "/api/auth/refresh" -Body $refreshBody -Description "Refresh token"
if ($refreshResult) {
    $token = $refreshResult.access_token
    $authHeaders = @{ Authorization = "Bearer $token" }
}

Write-Host ""
Write-Host "2. Chat Operations" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

# Create chat
$chatBody = @{ name = "Performance Test Chat"; type = "group" } | ConvertTo-Json
$chatResult = Invoke-TimedRequest -Method "POST" -Path "/api/chats" -Headers $authHeaders -Body $chatBody -Description "Create chat"
$chatId = $chatResult.id

# Get chats list
Invoke-TimedRequest -Method "GET" -Path "/api/chats?page=1&count=20" -Headers $authHeaders -Description "Get chats list"

# Get single chat
Invoke-TimedRequest -Method "GET" -Path "/api/chats/$chatId" -Headers $authHeaders -Description "Get chat details"

Write-Host ""
Write-Host "3. Message Operations (10 messages)" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

# Send messages
$messageIds = @()
for ($i = 1; $i -le 10; $i++) {
    $msgBody = @{ content = "Test message #$i - $(Get-Date -Format 'HH:mm:ss.fff')" } | ConvertTo-Json
    $msgResult = Invoke-TimedRequest -Method "POST" -Path "/api/chats/$chatId/messages" -Headers $authHeaders -Body $msgBody -Description "Send message #$i"
    if ($msgResult) {
        $messageIds += $msgResult.id
    }
}

# Get messages
Invoke-TimedRequest -Method "GET" -Path "/api/chats/$chatId/messages?page=1&count=50" -Headers $authHeaders -Description "Get messages"

Write-Host ""
Write-Host "4. Update/Delete Operations" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

if ($messageIds.Count -gt 0) {
    # Update message
    $updateBody = @{ content = "Updated message content" } | ConvertTo-Json
    Invoke-TimedRequest -Method "PUT" -Path "/api/chats/messages/$($messageIds[0])" -Headers $authHeaders -Body $updateBody -Description "Update message"

    # Add reaction
    $reactionBody = @{ emoji = "thumbs_up" } | ConvertTo-Json
    Invoke-TimedRequest -Method "POST" -Path "/api/chats/messages/$($messageIds[1])/reactions" -Headers $authHeaders -Body $reactionBody -Description "Add reaction"

    # Delete message
    Invoke-TimedRequest -Method "DELETE" -Path "/api/chats/messages/$($messageIds[-1])" -Headers $authHeaders -Description "Delete message"
}

Write-Host ""
Write-Host "5. Rapid Read Operations (20 reads)" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

for ($i = 1; $i -le 20; $i++) {
    Invoke-TimedRequest -Method "GET" -Path "/api/chats/$chatId/messages" -Headers $authHeaders -Description "Rapid read #$i"
}

Write-Host ""
Write-Host "6. Presence Operations" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

# Connect (simulate WebSocket connection)
$connectionId = "conn_$timestamp"
$connectBody = @{ connection_id = $connectionId } | ConvertTo-Json
Invoke-TimedRequest -Method "POST" -Path "/api/presence/connect" -Headers $authHeaders -Body $connectBody -Description "Presence connect"

# Set status to available
$statusBody = @{ status = "available" } | ConvertTo-Json
Invoke-TimedRequest -Method "PUT" -Path "/api/presence/status" -Headers $authHeaders -Body $statusBody -Description "Set status: available"

# Get my presence status
Invoke-TimedRequest -Method "GET" -Path "/api/presence/status" -Headers $authHeaders -Description "Get my presence"

# Change status to busy
$busyBody = @{ status = "busy" } | ConvertTo-Json
Invoke-TimedRequest -Method "PUT" -Path "/api/presence/status" -Headers $authHeaders -Body $busyBody -Description "Set status: busy"

# Change status to away
$awayBody = @{ status = "away" } | ConvertTo-Json
Invoke-TimedRequest -Method "PUT" -Path "/api/presence/status" -Headers $authHeaders -Body $awayBody -Description "Set status: away"

# Change status to dnd
$dndBody = @{ status = "dnd" } | ConvertTo-Json
Invoke-TimedRequest -Method "PUT" -Path "/api/presence/status" -Headers $authHeaders -Body $dndBody -Description "Set status: dnd"

# Get presence again
Invoke-TimedRequest -Method "GET" -Path "/api/presence/status" -Headers $authHeaders -Description "Verify status"

# Disconnect
$disconnectBody = @{ connection_id = $connectionId } | ConvertTo-Json
Invoke-TimedRequest -Method "POST" -Path "/api/presence/disconnect" -Headers $authHeaders -Body $disconnectBody -Description "Presence disconnect"

Write-Host ""
Write-Host "7. Cleanup" -ForegroundColor White
Write-Host "-" * 40 -ForegroundColor Gray

# Delete chat
Invoke-TimedRequest -Method "DELETE" -Path "/api/chats/$chatId" -Headers $authHeaders -Description "Delete chat"

# Logout
$logoutBody = @{ refresh_token = $refreshToken } | ConvertTo-Json
Invoke-TimedRequest -Method "POST" -Path "/api/auth/logout" -Headers $authHeaders -Body $logoutBody -Description "Logout"

# ==================== RESULTS ====================

$testEndTime = Get-Date
$testDuration = ($testEndTime - $testStartTime).TotalSeconds

Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  TEST RESULTS" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

# Calculate statistics
$successful = $results | Where-Object { $_.Success }
$failed = $results | Where-Object { -not $_.Success }
$durations = $successful | ForEach-Object { $_.Duration } | Sort-Object

$excellent = ($successful | Where-Object { $_.Rating -eq "EXCELLENT" }).Count
$good = ($successful | Where-Object { $_.Rating -eq "GOOD" }).Count
$acceptable = ($successful | Where-Object { $_.Rating -eq "ACCEPTABLE" }).Count
$slow = ($successful | Where-Object { $_.Rating -eq "SLOW" }).Count

$totalRequests = $results.Count
$avgDuration = if ($durations.Count -gt 0) { [math]::Round(($durations | Measure-Object -Average).Average) } else { 0 }
$minDuration = if ($durations.Count -gt 0) { $durations[0] } else { 0 }
$maxDuration = if ($durations.Count -gt 0) { $durations[-1] } else { 0 }
$p50 = if ($durations.Count -gt 0) { $durations[[math]::Floor($durations.Count * 0.5)] } else { 0 }
$p90 = if ($durations.Count -gt 0) { $durations[[math]::Floor($durations.Count * 0.9)] } else { 0 }
$p95 = if ($durations.Count -gt 0) { $durations[[math]::Floor($durations.Count * 0.95)] } else { 0 }

Write-Host "  Test Duration: $([math]::Round($testDuration, 2))s" -ForegroundColor Gray
Write-Host "  Total Requests: $totalRequests" -ForegroundColor White
Write-Host ""

Write-Host "  REQUEST DISTRIBUTION:" -ForegroundColor White
Write-Host "    Successful: $($successful.Count)" -ForegroundColor Green
Write-Host "    Failed:     $($failed.Count)" -ForegroundColor $(if ($failed.Count -gt 0) { "Red" } else { "Green" })
Write-Host ""
Write-Host "    [EXCELLENT] < 100ms:  $excellent ($([math]::Round($excellent/$totalRequests*100, 1))%)" -ForegroundColor Green
Write-Host "    [GOOD]      < 300ms:  $good ($([math]::Round($good/$totalRequests*100, 1))%)" -ForegroundColor Cyan
Write-Host "    [ACCEPTABLE]< 1s:     $acceptable ($([math]::Round($acceptable/$totalRequests*100, 1))%)" -ForegroundColor Yellow
Write-Host "    [SLOW]      >= 1s:    $slow ($([math]::Round($slow/$totalRequests*100, 1))%)" -ForegroundColor Red

Write-Host ""
Write-Host "  TIMING METRICS:" -ForegroundColor White
Write-Host "    Average: ${avgDuration}ms"
Write-Host "    Min:     ${minDuration}ms"
Write-Host "    Max:     ${maxDuration}ms"
Write-Host "    P50:     ${p50}ms"
Write-Host "    P90:     ${p90}ms"
Write-Host "    P95:     ${p95}ms"

if ($slow -gt 0) {
    Write-Host ""
    Write-Host "  SLOW REQUESTS:" -ForegroundColor Yellow
    $results | Where-Object { $_.Rating -eq "SLOW" } | ForEach-Object {
        Write-Host "    - $($_.Description): $($_.Duration)ms" -ForegroundColor Yellow
    }
}

if ($failed.Count -gt 0) {
    Write-Host ""
    Write-Host "  FAILED REQUESTS:" -ForegroundColor Red
    $failed | ForEach-Object {
        Write-Host "    - $($_.Description): $($_.Error)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "-" * 70 -ForegroundColor Gray

$slowPercent = $slow / $totalRequests * 100
if ($failed.Count -gt 0) {
    Write-Host "  VERDICT: FAIL - $($failed.Count) failed request(s)" -ForegroundColor Red
    exit 1
} elseif ($slowPercent -gt 10) {
    Write-Host "  VERDICT: WARNING - ${slowPercent}% slow requests" -ForegroundColor Yellow
} elseif ($slow -gt 0) {
    Write-Host "  VERDICT: OK - $slow slow request(s) ($([math]::Round($slowPercent, 1))%)" -ForegroundColor Yellow
} else {
    Write-Host "  VERDICT: EXCELLENT - All requests within limits" -ForegroundColor Green
}

Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

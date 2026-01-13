# Fault Tolerance Test for Centrifugo
# Tests that the system continues to work when a Centrifugo instance fails

param(
    [string]$ApiUrl = "http://127.0.0.1:8888"
)

$ErrorActionPreference = "Continue"

Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  CENTRIFUGO FAULT TOLERANCE TEST" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

# Generate unique test user
$timestamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
$random = -join ((97..122) | Get-Random -Count 6 | ForEach-Object { [char]$_ })
$username = "fault_${timestamp}_${random}"
$email = "${username}@test.local"
$password = "TestPass123!"

Write-Host "1. SETUP - Register user and create chat" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

# Register
try {
    $regBody = @{ username = $username; email = $email; password = $password } | ConvertTo-Json
    $reg = Invoke-RestMethod -Uri "$ApiUrl/api/auth/register" -Method Post -ContentType "application/json" -Body $regBody
    $token = $reg.access_token
    $headers = @{ Authorization = "Bearer $token" }
    Write-Host "  [OK] User registered: $username" -ForegroundColor Green
} catch {
    Write-Host "  [FAIL] Registration failed: $_" -ForegroundColor Red
    exit 1
}

# Create chat
try {
    $chatBody = @{ name = "Fault Tolerance Test Chat"; type = "group" } | ConvertTo-Json
    $chat = Invoke-RestMethod -Uri "$ApiUrl/api/chats" -Method Post -ContentType "application/json" -Headers $headers -Body $chatBody
    $chatId = $chat.id
    Write-Host "  [OK] Chat created: $chatId" -ForegroundColor Green
} catch {
    Write-Host "  [FAIL] Chat creation failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "2. TEST BEFORE FAILURE - Send messages" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

# Send 5 messages before stopping centrifugo
for ($i = 1; $i -le 5; $i++) {
    try {
        $msgBody = @{ content = "Message BEFORE failure #$i" } | ConvertTo-Json
        $msg = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages" -Method Post -ContentType "application/json" -Headers $headers -Body $msgBody
        Write-Host "  [OK] Message #$i sent" -ForegroundColor Green
    } catch {
        Write-Host "  [FAIL] Message #$i failed: $_" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "3. SIMULATE FAILURE - Stop one Centrifugo instance" -ForegroundColor Yellow
Write-Host "-" * 50 -ForegroundColor Gray

# Get centrifugo container names
$containers = docker ps --filter "name=centrifugo" --format "{{.Names}}" | Where-Object { $_ -match "centrifugo" }
Write-Host "  Running Centrifugo instances:" -ForegroundColor Gray
$containers | ForEach-Object { Write-Host "    - $_" -ForegroundColor Gray }

if ($containers.Count -ge 1) {
    $targetContainer = $containers[0]
    Write-Host "  Stopping: $targetContainer" -ForegroundColor Yellow
    docker stop $targetContainer | Out-Null
    Write-Host "  [OK] Container stopped" -ForegroundColor Yellow

    # Wait a moment for the system to detect the failure
    Start-Sleep -Seconds 2
} else {
    Write-Host "  [SKIP] No centrifugo containers found" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "4. TEST DURING FAILURE - Send messages (should use retry)" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

$successCount = 0
$failCount = 0

for ($i = 1; $i -le 10; $i++) {
    $startTime = Get-Date
    try {
        $msgBody = @{ content = "Message DURING failure #$i" } | ConvertTo-Json
        $msg = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages" -Method Post -ContentType "application/json" -Headers $headers -Body $msgBody -TimeoutSec 30
        $duration = ((Get-Date) - $startTime).TotalMilliseconds
        Write-Host "  [OK] Message #$i sent (${duration}ms)" -ForegroundColor Green
        $successCount++
    } catch {
        $duration = ((Get-Date) - $startTime).TotalMilliseconds
        Write-Host "  [FAIL] Message #$i failed (${duration}ms): $_" -ForegroundColor Red
        $failCount++
    }
    Start-Sleep -Milliseconds 200
}

Write-Host ""
Write-Host "5. RESTORE - Start the stopped instance" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

if ($targetContainer) {
    Write-Host "  Starting: $targetContainer" -ForegroundColor Cyan
    docker start $targetContainer | Out-Null
    Write-Host "  [OK] Container started" -ForegroundColor Green

    # Wait for it to become healthy
    Write-Host "  Waiting for container to be healthy..." -ForegroundColor Gray
    Start-Sleep -Seconds 5
}

Write-Host ""
Write-Host "6. TEST AFTER RECOVERY - Send messages" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

for ($i = 1; $i -le 5; $i++) {
    try {
        $msgBody = @{ content = "Message AFTER recovery #$i" } | ConvertTo-Json
        $msg = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages" -Method Post -ContentType "application/json" -Headers $headers -Body $msgBody
        Write-Host "  [OK] Message #$i sent" -ForegroundColor Green
        $successCount++
    } catch {
        Write-Host "  [FAIL] Message #$i failed: $_" -ForegroundColor Red
        $failCount++
    }
}

Write-Host ""
Write-Host "7. VERIFY - Check all messages were saved" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

try {
    $messages = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages?count=100" -Method Get -Headers $headers
    $totalMessages = $messages.messages.Count
    Write-Host "  Total messages in chat: $totalMessages" -ForegroundColor Cyan

    $beforeCount = ($messages.messages | Where-Object { $_.content -like "*BEFORE*" }).Count
    $duringCount = ($messages.messages | Where-Object { $_.content -like "*DURING*" }).Count
    $afterCount = ($messages.messages | Where-Object { $_.content -like "*AFTER*" }).Count

    Write-Host "    - Before failure: $beforeCount" -ForegroundColor Gray
    Write-Host "    - During failure: $duringCount" -ForegroundColor Gray
    Write-Host "    - After recovery: $afterCount" -ForegroundColor Gray
} catch {
    Write-Host "  [FAIL] Could not fetch messages: $_" -ForegroundColor Red
}

# Results
Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  RESULTS" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""
Write-Host "  Messages sent successfully: $successCount" -ForegroundColor Green
Write-Host "  Messages failed: $failCount" -ForegroundColor $(if ($failCount -gt 0) { "Red" } else { "Green" })
Write-Host ""

if ($failCount -eq 0) {
    Write-Host "  VERDICT: EXCELLENT - All messages delivered despite failure!" -ForegroundColor Green
} elseif ($failCount -le 2) {
    Write-Host "  VERDICT: GOOD - Minor message loss during failover" -ForegroundColor Yellow
} else {
    Write-Host "  VERDICT: NEEDS IMPROVEMENT - Significant message loss" -ForegroundColor Red
}

Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

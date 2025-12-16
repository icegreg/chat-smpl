# Reconnection Scenario Test
# Simulates what happens when a device loses internet for 30 seconds
# and then reconnects

param(
    [string]$ApiUrl = "http://127.0.0.1:8888"
)

$ErrorActionPreference = "Continue"

Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  RECONNECTION SCENARIO TEST" -ForegroundColor Cyan
Write-Host "  (Multiple devices, one goes offline)" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

# Generate unique test users
$timestamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
$random1 = -join ((97..122) | Get-Random -Count 4 | ForEach-Object { [char]$_ })
$random2 = -join ((97..122) | Get-Random -Count 4 | ForEach-Object { [char]$_ })

$user1 = "recon1_${timestamp}_${random1}"
$user2 = "recon2_${timestamp}_${random2}"
$password = "TestPass123!"

Write-Host "1. SETUP - Register users and create shared chat" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

# Register User 1
try {
    $regBody = @{ username = $user1; email = "${user1}@test.local"; password = $password } | ConvertTo-Json
    $reg1 = Invoke-RestMethod -Uri "$ApiUrl/api/auth/register" -Method Post -ContentType "application/json" -Body $regBody
    $token1 = $reg1.access_token
    $userId1 = $reg1.user.id
    $headers1 = @{ Authorization = "Bearer $token1" }
    Write-Host "  [OK] User 1 registered: $user1" -ForegroundColor Green
} catch {
    Write-Host "  [FAIL] User 1 registration failed: $_" -ForegroundColor Red
    exit 1
}

# Register User 2
try {
    $regBody = @{ username = $user2; email = "${user2}@test.local"; password = $password } | ConvertTo-Json
    $reg2 = Invoke-RestMethod -Uri "$ApiUrl/api/auth/register" -Method Post -ContentType "application/json" -Body $regBody
    $token2 = $reg2.access_token
    $userId2 = $reg2.user.id
    $headers2 = @{ Authorization = "Bearer $token2" }
    Write-Host "  [OK] User 2 registered: $user2" -ForegroundColor Green
} catch {
    Write-Host "  [FAIL] User 2 registration failed: $_" -ForegroundColor Red
    exit 1
}

# Create shared chat
try {
    $chatBody = @{ name = "Reconnection Test Chat"; type = "group"; participant_ids = @($userId2) } | ConvertTo-Json
    $chat = Invoke-RestMethod -Uri "$ApiUrl/api/chats" -Method Post -ContentType "application/json" -Headers $headers1 -Body $chatBody
    $chatId = $chat.id
    Write-Host "  [OK] Chat created: $chatId" -ForegroundColor Green
} catch {
    Write-Host "  [FAIL] Chat creation failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "2. USER 1 SENDS INITIAL MESSAGES" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

# User 1 sends some messages
$initialMessages = @()
for ($i = 1; $i -le 3; $i++) {
    try {
        $msgBody = @{ content = "Initial message #$i from User 1" } | ConvertTo-Json
        $msg = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages" -Method Post -ContentType "application/json" -Headers $headers1 -Body $msgBody
        $initialMessages += $msg
        Write-Host "  [OK] Message #$i sent (seq_num: $($msg.seq_num))" -ForegroundColor Green
    } catch {
        Write-Host "  [FAIL] Message #$i failed: $_" -ForegroundColor Red
    }
}

# Track last seq_num for User 2
$lastSeqNumUser2 = $initialMessages[-1].seq_num
Write-Host "  User 2 last known seq_num: $lastSeqNumUser2" -ForegroundColor Cyan

Write-Host ""
Write-Host "3. SIMULATE USER 2 GOING OFFLINE (30 seconds)" -ForegroundColor Yellow
Write-Host "-" * 50 -ForegroundColor Gray
Write-Host "  User 2 device loses internet connection..." -ForegroundColor Yellow

# While User 2 is "offline", User 1 sends more messages
Write-Host ""
Write-Host "4. USER 1 SENDS MESSAGES WHILE USER 2 IS OFFLINE" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

$offlineMessages = @()
for ($i = 1; $i -le 5; $i++) {
    try {
        $msgBody = @{ content = "Message while User 2 offline #$i" } | ConvertTo-Json
        $msg = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages" -Method Post -ContentType "application/json" -Headers $headers1 -Body $msgBody
        $offlineMessages += $msg
        Write-Host "  [OK] Message #$i sent (seq_num: $($msg.seq_num))" -ForegroundColor Green
        Start-Sleep -Milliseconds 500
    } catch {
        Write-Host "  [FAIL] Message #$i failed: $_" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "5. USER 2 RECONNECTS - SYNC VIA API" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray
Write-Host "  User 2 device reconnects after 30 seconds..." -ForegroundColor Cyan
Write-Host "  Syncing messages after seq_num: $lastSeqNumUser2" -ForegroundColor Cyan

# User 2 syncs missed messages via API (simulating what the client does)
try {
    $syncResult = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages/sync?after_seq=$lastSeqNumUser2&limit=100" -Method Get -Headers $headers2
    $syncedMessages = $syncResult.messages

    Write-Host ""
    Write-Host "  SYNC RESULTS:" -ForegroundColor Cyan
    Write-Host "  Messages recovered: $($syncedMessages.Count)" -ForegroundColor $(if ($syncedMessages.Count -eq 5) { "Green" } else { "Yellow" })
    Write-Host "  Has more: $($syncResult.has_more)" -ForegroundColor Gray

    if ($syncedMessages.Count -gt 0) {
        Write-Host ""
        Write-Host "  Recovered messages:" -ForegroundColor Gray
        foreach ($msg in $syncedMessages) {
            Write-Host "    - seq_num $($msg.seq_num): $($msg.content)" -ForegroundColor Gray
        }
    }
} catch {
    Write-Host "  [FAIL] Sync failed: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "6. VERIFY - Check message counts" -ForegroundColor White
Write-Host "-" * 50 -ForegroundColor Gray

try {
    $allMessages = Invoke-RestMethod -Uri "$ApiUrl/api/chats/$chatId/messages?count=100" -Method Get -Headers $headers2
    $totalCount = $allMessages.messages.Count

    Write-Host "  Total messages in chat: $totalCount" -ForegroundColor Cyan
    Write-Host "  Expected: $(3 + 5) (3 initial + 5 while offline)" -ForegroundColor Gray

    $initialCount = ($allMessages.messages | Where-Object { $_.content -like "*Initial*" }).Count
    $offlineCount = ($allMessages.messages | Where-Object { $_.content -like "*offline*" }).Count

    Write-Host "    - Initial messages: $initialCount" -ForegroundColor Gray
    Write-Host "    - Offline messages: $offlineCount" -ForegroundColor Gray
} catch {
    Write-Host "  [FAIL] Verification failed: $_" -ForegroundColor Red
}

# Results
Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host "  RESULTS" -ForegroundColor Cyan
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

if ($syncedMessages.Count -eq 5) {
    Write-Host "  VERDICT: SUCCESS - All 5 missed messages recovered!" -ForegroundColor Green
    Write-Host ""
    Write-Host "  This demonstrates that when a device goes offline:" -ForegroundColor Gray
    Write-Host "    1. Messages continue to be stored in the database" -ForegroundColor Gray
    Write-Host "    2. seq_num tracking allows precise sync" -ForegroundColor Gray
    Write-Host "    3. Client can recover ALL missed messages via sync API" -ForegroundColor Gray
} else {
    Write-Host "  VERDICT: PARTIAL - Only $($syncedMessages.Count)/5 messages recovered" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=" * 70 -ForegroundColor Cyan
Write-Host ""

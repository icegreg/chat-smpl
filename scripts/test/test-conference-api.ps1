# Login first
$loginBody = @{
    email = "testuser123@test.com"
    password = "Password123"
} | ConvertTo-Json -Compress

$response = Invoke-RestMethod -Uri "http://localhost:8888/api/auth/login" -Method Post -ContentType "application/json" -Body $loginBody
$accessToken = $response.access_token

Write-Host "Access token obtained"

$headers = @{
    Authorization = "Bearer $accessToken"
}

# Create a test chat
Write-Host ""
Write-Host "Creating test chat..."
$chatBody = @{
    name = "Test Conference Chat"
    is_group = $true
} | ConvertTo-Json -Compress

try {
    $chat = Invoke-RestMethod -Uri "http://localhost:8888/api/chats" -Method Post -ContentType "application/json" -Headers $headers -Body $chatBody
    Write-Host "Chat created: $($chat.id)"
    $chatId = $chat.id
} catch {
    Write-Host "Chat creation error: $_"
    exit 1
}

# Get conference history for the chat (should be empty)
Write-Host ""
Write-Host "Testing GET /api/voice/chats/{chatId}/conferences/history..."
try {
    $history = Invoke-RestMethod -Uri "http://localhost:8888/api/voice/chats/$chatId/conferences/history" -Method Get -Headers $headers
    Write-Host "Success! Conference history:"
    $history | ConvertTo-Json -Depth 5
} catch {
    Write-Host "Error: $_"
    Write-Host "Response: $($_.Exception.Response)"
}

# Test getting conference by ID (use one of the existing conference IDs)
Write-Host ""
Write-Host "Testing GET /api/voice/conferences/{conferenceId}/history..."
$confId = "2b139fcb-5512-45dc-b224-a8fee2f7fab4"  # One of the existing conferences
try {
    $confHistory = Invoke-RestMethod -Uri "http://localhost:8888/api/voice/conferences/$confId/history" -Method Get -Headers $headers
    Write-Host "Success! Conference details:"
    $confHistory | ConvertTo-Json -Depth 5
} catch {
    Write-Host "Error: $($_.Exception.Message)"
}

# Test getting conference messages
Write-Host ""
Write-Host "Testing GET /api/voice/conferences/{conferenceId}/messages..."
try {
    $messages = Invoke-RestMethod -Uri "http://localhost:8888/api/voice/conferences/$confId/messages" -Method Get -Headers $headers
    Write-Host "Success! Conference messages:"
    $messages | ConvertTo-Json -Depth 5
} catch {
    Write-Host "Error: $($_.Exception.Message)"
}

# Test getting moderator actions
Write-Host ""
Write-Host "Testing GET /api/voice/conferences/{conferenceId}/moderator-actions..."
try {
    $actions = Invoke-RestMethod -Uri "http://localhost:8888/api/voice/conferences/$confId/moderator-actions" -Method Get -Headers $headers
    Write-Host "Success! Moderator actions:"
    $actions | ConvertTo-Json -Depth 5
} catch {
    Write-Host "Error: $($_.Exception.Message)"
}

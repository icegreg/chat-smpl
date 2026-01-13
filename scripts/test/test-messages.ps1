$timestamp = [int64](Get-Date -UFormat %s)
$reg = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/auth/register' -Method Post -ContentType 'application/json' -Body "{`"username`":`"msgtest$timestamp`",`"password`":`"test1234`",`"email`":`"msgtest$timestamp@example.com`",`"display_name`":`"Test User $timestamp`"}"
$token = $reg.access_token
Write-Host "Token obtained for user with display_name: Test User $timestamp"

# Create chat
$chat = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/chats' -Method Post -ContentType 'application/json' -Headers @{Authorization="Bearer $token"} -Body '{"name":"Message Test Chat","type":"group"}'
Write-Host "Chat ID: $($chat.id)"
$chatId = $chat.id

# Send message
$msg = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/chats/$chatId/messages" -Method Post -ContentType 'application/json' -Headers @{Authorization="Bearer $token"} -Body '{"content":"Hello world"}'
Write-Host "`nSent message:"
$msg | ConvertTo-Json -Depth 10

# Get messages
$msgs = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/chats/$chatId/messages" -Method Get -Headers @{Authorization="Bearer $token"}
Write-Host "`nGet messages response:"
$msgs | ConvertTo-Json -Depth 10

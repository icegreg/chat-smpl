# Test API to check ws_url
$ErrorActionPreference = "Stop"

# Register
$regBody = @{
    email = "apitest_$(Get-Date -Format 'yyyyMMddHHmmss')@test.com"
    username = "apitest_$(Get-Date -Format 'yyyyMMddHHmmss')"
    password = "test1234"
} | ConvertTo-Json

$reg = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/auth/register' -Method POST -ContentType 'application/json' -Body $regBody
$token = $reg.access_token
Write-Host "Registered user: $($reg.user.username)"

# Create chat
$chatBody = @{ name = "TestCallChat"; type = "group" } | ConvertTo-Json
$chat = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/chats' -Method POST -ContentType 'application/json' -Headers @{Authorization = "Bearer $token"} -Body $chatBody
Write-Host "Created chat: $($chat.id)"

# Start chat call
$callResult = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/voice/chats/$($chat.id)/call" -Method POST -ContentType 'application/json' -Headers @{Authorization = "Bearer $token"}
Write-Host "`nCall result:"
Write-Host "  conference.id: $($callResult.conference.id)"
Write-Host "  conference.freeswitch_name: $($callResult.conference.freeswitch_name)"
Write-Host "  credentials.ws_url: $($callResult.credentials.ws_url)"
Write-Host "  credentials.login: $($callResult.credentials.login)"

$body = @{
    username = "testuser123"
    email = "testuser123@test.com"
    password = "Password123"
} | ConvertTo-Json -Compress

Write-Host "Register body: $body"

try {
    $response = Invoke-RestMethod -Uri "http://localhost:8888/api/auth/register" -Method Post -ContentType "application/json" -Body $body
    Write-Host "Register response:"
    $response | ConvertTo-Json
    $accessToken = $response.access_token
} catch {
    Write-Host "Register error: $_"
    # Try login
    $loginBody = @{
        email = "testuser123@test.com"
        password = "Password123"
    } | ConvertTo-Json -Compress

    Write-Host "Login body: $loginBody"
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8888/api/auth/login" -Method Post -ContentType "application/json" -Body $loginBody
        Write-Host "Login response:"
        $response | ConvertTo-Json
        $accessToken = $response.access_token
    } catch {
        Write-Host "Login error: $_"
        exit 1
    }
}

Write-Host "Access token: $accessToken"

# Test conference history endpoint
Write-Host ""
Write-Host "Testing conference history endpoint..."
$headers = @{
    Authorization = "Bearer $accessToken"
}

try {
    # Get a chat ID first
    $chats = Invoke-RestMethod -Uri "http://localhost:8888/api/chats" -Method Get -Headers $headers
    Write-Host "Chats: $($chats | ConvertTo-Json -Depth 3)"

    if ($chats -and $chats.data -and $chats.data.Length -gt 0) {
        $chatId = $chats.data[0].id
        Write-Host "Testing conference history for chat: $chatId"

        $historyUrl = "http://localhost:8888/api/voice/chats/$chatId/conferences/history"
        Write-Host "URL: $historyUrl"

        $history = Invoke-RestMethod -Uri $historyUrl -Method Get -Headers $headers
        Write-Host "Conference history response:"
        $history | ConvertTo-Json -Depth 5
    } else {
        Write-Host "No chats found"
    }
} catch {
    Write-Host "Error: $_"
}

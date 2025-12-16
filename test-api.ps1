$reg = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/auth/register' -Method Post -ContentType 'application/json' -Body '{"username":"testfile123","password":"test1234","email":"testfile123@example.com"}'
$token = $reg.access_token
Write-Host "Token: $($token.Substring(0, 50))..."

# Create chat
$chat = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/chats' -Method Post -ContentType 'application/json' -Headers @{Authorization="Bearer $token"} -Body '{"name":"Test Chat","type":"group"}'
Write-Host "Chat ID: $($chat.id)"

# Upload file
$filePath = [System.IO.Path]::GetTempFileName()
'test content' | Out-File -FilePath $filePath -NoNewline -Encoding UTF8
$fileBytes = [System.IO.File]::ReadAllBytes($filePath)

$boundary = [System.Guid]::NewGuid().ToString()
$CRLF = "`r`n"
$bodyLines = @(
    "--$boundary",
    "Content-Disposition: form-data; name=`"file`"; filename=`"test.txt`"",
    "Content-Type: text/plain",
    "",
    "test content",
    "--$boundary--",
    ""
) -join $CRLF

try {
    $uploadResult = Invoke-RestMethod -Uri 'http://127.0.0.1:8888/api/files/upload' -Method Post -ContentType "multipart/form-data; boundary=$boundary" -Headers @{Authorization="Bearer $token"} -Body $bodyLines
    Write-Host "Upload result:"
    $uploadResult | ConvertTo-Json
} catch {
    Write-Host "Upload error: $_"
    exit 1
}

# Send message with file
$chatId = $chat.id
$linkId = $uploadResult.link_id
$messageBody = "{`"content`":`"Message with file`",`"file_link_ids`":[`"$linkId`"]}"
Write-Host "Sending message with body: $messageBody"
$msg = Invoke-RestMethod -Uri "http://127.0.0.1:8888/api/chats/$chatId/messages" -Method Post -ContentType 'application/json' -Headers @{Authorization="Bearer $token"} -Body $messageBody
Write-Host "Message result:"
$msg | ConvertTo-Json -Depth 10

$ErrorActionPreference = "Stop"
$baseUrl = "http://127.0.0.1:8888"

Write-Host "=== Files API Test ===" -ForegroundColor Cyan
Write-Host ""

# Helper function for API calls
function Invoke-Api {
    param(
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Headers = @{},
        [object]$Body = $null,
        [string]$ContentType = "application/json",
        [switch]$ExpectError
    )

    $uri = "$baseUrl$Endpoint"
    $params = @{
        Method = $Method
        Uri = $uri
        Headers = $Headers
        ContentType = $ContentType
    }

    if ($Body -and $Method -ne "GET") {
        if ($ContentType -eq "application/json") {
            $params.Body = ($Body | ConvertTo-Json -Depth 10)
        } else {
            $params.Body = $Body
        }
    }

    try {
        $response = Invoke-RestMethod @params
        return @{ Success = $true; Data = $response; StatusCode = 200 }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        $errorBody = $null
        try {
            $stream = $_.Exception.Response.GetResponseStream()
            $reader = New-Object System.IO.StreamReader($stream)
            $errorBody = $reader.ReadToEnd()
        } catch {}

        if ($ExpectError) {
            return @{ Success = $false; StatusCode = $statusCode; Error = $errorBody }
        }

        Write-Host "  ERROR: $Method $Endpoint" -ForegroundColor Red
        Write-Host "  Status: $statusCode" -ForegroundColor Red
        Write-Host "  Response: $errorBody" -ForegroundColor Red
        return @{ Success = $false; StatusCode = $statusCode; Error = $errorBody }
    }
}

# Generate unique username
$timestamp = Get-Date -Format "yyyyMMddHHmmss"
$username = "filetest_$timestamp"
$email = "$username@test.local"
$password = "Test123456!"

Write-Host "1. Register test user: $username" -ForegroundColor Yellow
$registerResult = Invoke-Api -Method POST -Endpoint "/api/auth/register" -Body @{
    username = $username
    email = $email
    password = $password
    display_name = "File Test User"
}

if (-not $registerResult.Success) {
    Write-Host "  FAILED to register user" -ForegroundColor Red
    exit 1
}
Write-Host "  OK - User registered" -ForegroundColor Green
$accessToken = $registerResult.Data.access_token
$userId = $registerResult.Data.user.id

# Auth headers
$authHeaders = @{ "Authorization" = "Bearer $accessToken" }

Write-Host ""
Write-Host "2. Create test file" -ForegroundColor Yellow
$testContent = "This is a test file content for API testing. Timestamp: $timestamp"
$testFileName = "test_$timestamp.txt"
$tempFile = [System.IO.Path]::GetTempFileName()
Set-Content -Path $tempFile -Value $testContent -NoNewline

Write-Host ""
Write-Host "3. Upload file via multipart form" -ForegroundColor Yellow

# Create multipart form data manually
$boundary = [System.Guid]::NewGuid().ToString()
$LF = "`r`n"

$fileBytes = [System.IO.File]::ReadAllBytes($tempFile)
$fileEnc = [System.Text.Encoding]::GetEncoding("iso-8859-1").GetString($fileBytes)

$bodyLines = @(
    "--$boundary",
    "Content-Disposition: form-data; name=`"file`"; filename=`"$testFileName`"",
    "Content-Type: text/plain",
    "",
    $fileEnc,
    "--$boundary--",
    ""
) -join $LF

try {
    $uploadResponse = Invoke-RestMethod -Method POST -Uri "$baseUrl/api/files/upload" `
        -Headers @{ "Authorization" = "Bearer $accessToken" } `
        -ContentType "multipart/form-data; boundary=$boundary" `
        -Body $bodyLines

    Write-Host "  OK - File uploaded" -ForegroundColor Green
    Write-Host "  Link ID: $($uploadResponse.link_id)" -ForegroundColor Gray
    Write-Host "  File ID: $($uploadResponse.file_id)" -ForegroundColor Gray
    Write-Host "  Filename: $($uploadResponse.filename)" -ForegroundColor Gray
    Write-Host "  Original: $($uploadResponse.original_filename)" -ForegroundColor Gray
    Write-Host "  Size: $($uploadResponse.size) bytes" -ForegroundColor Gray
    Write-Host "  Content-Type: $($uploadResponse.content_type)" -ForegroundColor Gray

    $linkId = $uploadResponse.link_id
    $fileId = $uploadResponse.file_id
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "  FAILED - Status: $statusCode" -ForegroundColor Red

    try {
        $stream = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($stream)
        $errorBody = $reader.ReadToEnd()
        Write-Host "  Response: $errorBody" -ForegroundColor Red
    } catch {
        Write-Host "  Could not read error response" -ForegroundColor Red
    }

    Write-Host ""
    Write-Host "=== Checking Files Service logs ===" -ForegroundColor Yellow
    docker logs chatapp-files --tail 20 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }

    Remove-Item $tempFile -ErrorAction SilentlyContinue
    exit 1
}

# Cleanup temp file
Remove-Item $tempFile -ErrorAction SilentlyContinue

if (-not $linkId) {
    Write-Host "  No link_id returned, cannot continue" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "4. Get file info" -ForegroundColor Yellow
$infoResult = Invoke-Api -Method GET -Endpoint "/api/files/$linkId/info" -Headers $authHeaders

if ($infoResult.Success) {
    Write-Host "  OK - Got file info" -ForegroundColor Green
    Write-Host "  Original filename: $($infoResult.Data.original_filename)" -ForegroundColor Gray
    Write-Host "  Size: $($infoResult.Data.size)" -ForegroundColor Gray
} else {
    Write-Host "  FAILED to get file info" -ForegroundColor Red
}

Write-Host ""
Write-Host "5. Download file" -ForegroundColor Yellow
try {
    $downloadResponse = Invoke-WebRequest -Method GET -Uri "$baseUrl/api/files/$linkId" `
        -Headers @{ "Authorization" = "Bearer $accessToken" }

    $downloadedContent = [System.Text.Encoding]::UTF8.GetString($downloadResponse.Content)

    if ($downloadedContent -eq $testContent) {
        Write-Host "  OK - Downloaded content matches" -ForegroundColor Green
    } else {
        Write-Host "  WARNING - Content mismatch" -ForegroundColor Yellow
        Write-Host "  Expected: $testContent" -ForegroundColor Gray
        Write-Host "  Got: $downloadedContent" -ForegroundColor Gray
    }
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "  FAILED - Status: $statusCode" -ForegroundColor Red
}

Write-Host ""
Write-Host "6. Test access without auth (should fail)" -ForegroundColor Yellow
$noAuthResult = Invoke-Api -Method GET -Endpoint "/api/files/$linkId" -ExpectError

if ($noAuthResult.StatusCode -eq 401) {
    Write-Host "  OK - Correctly denied (401)" -ForegroundColor Green
} else {
    Write-Host "  UNEXPECTED - Status: $($noAuthResult.StatusCode)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "7. Register second user and test access" -ForegroundColor Yellow
$username2 = "filetest2_$timestamp"
$registerResult2 = Invoke-Api -Method POST -Endpoint "/api/auth/register" -Body @{
    username = $username2
    email = "$username2@test.local"
    password = $password
    display_name = "File Test User 2"
}

if ($registerResult2.Success) {
    $accessToken2 = $registerResult2.Data.access_token
    $authHeaders2 = @{ "Authorization" = "Bearer $accessToken2" }

    Write-Host "  Second user registered" -ForegroundColor Gray

    # Try to access file as second user (should fail - no permission)
    $accessResult = Invoke-Api -Method GET -Endpoint "/api/files/$linkId" -Headers $authHeaders2 -ExpectError

    if ($accessResult.StatusCode -eq 403) {
        Write-Host "  OK - Second user correctly denied (403)" -ForegroundColor Green
    } else {
        Write-Host "  UNEXPECTED - Status: $($accessResult.StatusCode)" -ForegroundColor Yellow
    }
} else {
    Write-Host "  Could not register second user" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "8. Create a chat and send message with file" -ForegroundColor Yellow

# Create chat
$chatResult = Invoke-Api -Method POST -Endpoint "/api/chats" -Headers $authHeaders -Body @{
    title = "File Test Chat $timestamp"
    chat_type = "group"
}

if ($chatResult.Success) {
    $chatId = $chatResult.Data.id
    Write-Host "  Chat created: $chatId" -ForegroundColor Gray

    # Upload another file for the message
    $msgFileContent = "Message attachment content $timestamp"
    $msgFileName = "msg_attachment_$timestamp.txt"
    $tempFile2 = [System.IO.Path]::GetTempFileName()
    Set-Content -Path $tempFile2 -Value $msgFileContent -NoNewline

    $fileBytes2 = [System.IO.File]::ReadAllBytes($tempFile2)
    $fileEnc2 = [System.Text.Encoding]::GetEncoding("iso-8859-1").GetString($fileBytes2)

    $boundary2 = [System.Guid]::NewGuid().ToString()
    $bodyLines2 = @(
        "--$boundary2",
        "Content-Disposition: form-data; name=`"file`"; filename=`"$msgFileName`"",
        "Content-Type: text/plain",
        "",
        $fileEnc2,
        "--$boundary2--",
        ""
    ) -join $LF

    try {
        $uploadResponse2 = Invoke-RestMethod -Method POST -Uri "$baseUrl/api/files/upload" `
            -Headers @{ "Authorization" = "Bearer $accessToken" } `
            -ContentType "multipart/form-data; boundary=$boundary2" `
            -Body $bodyLines2

        $msgLinkId = $uploadResponse2.link_id
        Write-Host "  File for message uploaded: $msgLinkId" -ForegroundColor Gray

        # Send message with file
        $msgResult = Invoke-Api -Method POST -Endpoint "/api/chats/$chatId/messages" -Headers $authHeaders -Body @{
            content = "Test message with file"
            file_link_ids = @($msgLinkId)
        }

        if ($msgResult.Success) {
            Write-Host "  OK - Message with file sent" -ForegroundColor Green
            Write-Host "  Message ID: $($msgResult.Data.id)" -ForegroundColor Gray

            if ($msgResult.Data.file_attachments -and $msgResult.Data.file_attachments.Count -gt 0) {
                Write-Host "  Attachments: $($msgResult.Data.file_attachments.Count)" -ForegroundColor Gray
            }
        } else {
            Write-Host "  FAILED to send message with file" -ForegroundColor Red
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        Write-Host "  FAILED to upload file for message - Status: $statusCode" -ForegroundColor Red
    }

    Remove-Item $tempFile2 -ErrorAction SilentlyContinue
} else {
    Write-Host "  FAILED to create chat" -ForegroundColor Red
}

Write-Host ""
Write-Host "9. Delete file" -ForegroundColor Yellow
$deleteResult = Invoke-Api -Method DELETE -Endpoint "/api/files/$linkId" -Headers $authHeaders -ExpectError

if ($deleteResult.StatusCode -eq 204 -or $deleteResult.Success) {
    Write-Host "  OK - File deleted" -ForegroundColor Green
} else {
    Write-Host "  Status: $($deleteResult.StatusCode)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "10. Verify file is deleted (should return 404)" -ForegroundColor Yellow
$verifyResult = Invoke-Api -Method GET -Endpoint "/api/files/$linkId" -Headers $authHeaders -ExpectError

if ($verifyResult.StatusCode -eq 404) {
    Write-Host "  OK - File correctly not found (404)" -ForegroundColor Green
} else {
    Write-Host "  UNEXPECTED - Status: $($verifyResult.StatusCode)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "11. Test GET /api/files/chats/{chatId}/files endpoint" -ForegroundColor Yellow
if ($chatId) {
    $chatFilesResult = Invoke-Api -Method GET -Endpoint "/api/files/chats/$chatId/files" -Headers $authHeaders

    if ($chatFilesResult.Success) {
        Write-Host "  OK - Got chat files" -ForegroundColor Green
        Write-Host "  Files count: $($chatFilesResult.Data.files.Count)" -ForegroundColor Gray
        if ($chatFilesResult.Data.files -and $chatFilesResult.Data.files.Count -gt 0) {
            foreach ($file in $chatFilesResult.Data.files) {
                Write-Host "    - $($file.original_filename) ($($file.size) bytes)" -ForegroundColor Gray
            }
        }
    } else {
        Write-Host "  FAILED - Status: $($chatFilesResult.StatusCode)" -ForegroundColor Red
        Write-Host "  Error: $($chatFilesResult.Error)" -ForegroundColor Red
    }
} else {
    Write-Host "  SKIPPED - No chat ID available" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan

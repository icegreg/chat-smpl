$baseUrl = "http://127.0.0.1:8888"
$timestamp = Get-Date -Format "yyyyMMddHHmmssff"

Write-Host "=== Download Test ===" -ForegroundColor Cyan

# Register
$reg = Invoke-RestMethod -Method POST -Uri "$baseUrl/api/auth/register" -ContentType "application/json" -Body (@{
    username = "dltest_$timestamp"
    email = "dltest_$timestamp@test.local"
    password = "Test123456!"
    display_name = "Download Test"
} | ConvertTo-Json)

$token = $reg.access_token
Write-Host "Registered user, got token" -ForegroundColor Green

# Upload file
$boundary = [guid]::NewGuid().ToString()
$content = "Hello World Content $timestamp"
$body = "--$boundary`r`nContent-Disposition: form-data; name=`"file`"; filename=`"test.txt`"`r`nContent-Type: text/plain`r`n`r`n$content`r`n--$boundary--`r`n"

$upload = Invoke-RestMethod -Method POST -Uri "$baseUrl/api/files/upload" -Headers @{ Authorization = "Bearer $token" } -ContentType "multipart/form-data; boundary=$boundary" -Body $body
Write-Host "Uploaded file: $($upload.link_id)" -ForegroundColor Green

# Download
Write-Host "`nDownloading..." -ForegroundColor Yellow
try {
    $download = Invoke-WebRequest -Method GET -Uri "$baseUrl/api/files/$($upload.link_id)" -Headers @{ Authorization = "Bearer $token" }
    Write-Host "Status: $($download.StatusCode)" -ForegroundColor Green
    $downloadedContent = [System.Text.Encoding]::UTF8.GetString($download.Content)
    Write-Host "Content: $downloadedContent" -ForegroundColor Gray

    if ($downloadedContent -eq $content) {
        Write-Host "Content matches!" -ForegroundColor Green
    } else {
        Write-Host "Content mismatch!" -ForegroundColor Red
        Write-Host "Expected: $content" -ForegroundColor Gray
    }
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    Write-Host "Error Status: $statusCode" -ForegroundColor Red

    try {
        $stream = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($stream)
        Write-Host "Error Body: $($reader.ReadToEnd())" -ForegroundColor Red
    } catch {}
}

Write-Host "`n=== Test Complete ===" -ForegroundColor Cyan

$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location 'C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web'

$env:HEADLESS = 'false'
$env:BASE_URL = 'http://127.0.0.1:8888'

Write-Host "=== Message Forward E2E Test ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "This test verifies:" -ForegroundColor Yellow
Write-Host "1. User1 creates Chat1 with User2, sends message with file"
Write-Host "2. User1 creates Chat2 with User3, forwards message to Chat2"
Write-Host "3. User3 (Chat2 participant) CAN access forwarded file"
Write-Host "4. User2 (NOT in Chat2) CANNOT access forwarded file"
Write-Host "5. User2 CAN still access original file in Chat1"
Write-Host "6. User3 (NOT in Chat1) CANNOT access original file"
Write-Host ""

& "C:\Program Files\nodejs\npm.cmd" run test:e2e -- --grep "Message Forward"

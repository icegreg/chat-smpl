$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location 'C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web'

$env:HEADLESS = 'false'
$env:BASE_URL = 'http://127.0.0.1:8888'

Write-Host "=== Message Display E2E Test ===" -ForegroundColor Cyan
Write-Host ""

& "C:\Program Files\nodejs\npm.cmd" run test:e2e -- --grep "Message Display"

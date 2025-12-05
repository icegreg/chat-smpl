$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location 'C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web'

$env:USER_COUNT = '3'
$env:TEST_DURATION = '1'
$env:BASE_URL = 'http://127.0.0.1:8888'
$env:HEADLESS = 'false'

Write-Host "=== Quick Multi-User Chat E2E Test ===" -ForegroundColor Cyan
Write-Host "Users: 3"
Write-Host "Duration: 1 minute"
Write-Host ""

& "C:\Program Files\nodejs\npm.cmd" run test:e2e:multiuser

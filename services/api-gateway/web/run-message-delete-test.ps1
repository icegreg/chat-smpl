$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location 'C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web'

# Run in visible browser mode (not headless)
$env:HEADLESS = 'false'
$env:BASE_URL = 'http://127.0.0.1:8888'

Write-Host "=== Message Delete & Restore E2E Test ===" -ForegroundColor Cyan
Write-Host "Running in visible browser mode (HEADLESS=false)" -ForegroundColor Yellow
Write-Host ""

& "C:\Program Files\nodejs\npm.cmd" run test:e2e -- --grep "Message Delete"

$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
$env:HEADLESS = "true"
$env:BASE_URL = "http://127.0.0.1:8888"
Write-Host "Starting Voice Navigation E2E tests..."
& "C:\Program Files\nodejs\npx.cmd" mocha --require ts-node/register e2e-selenium/tests/voice-navigation.spec.ts --timeout 120000

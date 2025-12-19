$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
$env:HEADLESS = "true"
$env:BASE_URL = "http://127.0.0.1:8888"
Write-Host "Starting File Security E2E test..."
& "C:\Program Files\nodejs\npx.cmd" mocha --require ts-node/register e2e-selenium/tests/file-security.spec.ts --grep "File Security" --timeout 120000

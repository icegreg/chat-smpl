$env:HEADLESS = "false"
$env:BASE_URL = "http://127.0.0.1:8888"
$env:Path = "C:\Program Files\nodejs;C:\Users\ryabikov\AppData\Roaming\npm;" + $env:Path
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
& "C:\Program Files\nodejs\npx.cmd" mocha --require ts-node/register --grep "Reply and Forward" e2e-selenium/tests/reply-forward.spec.ts --timeout 120000 --exit

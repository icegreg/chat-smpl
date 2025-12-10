$env:HEADLESS = "false"
$env:BASE_URL = "http://127.0.0.1:8888"
$env:PATH = "C:\Program Files\nodejs;" + $env:PATH
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
& "C:\Program Files\nodejs\npm.cmd" run test:e2e -- --grep "Message Reply"

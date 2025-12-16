$env:PATH = "C:\Program Files\nodejs;" + $env:PATH
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web\e2e-selenium"
& "C:\Program Files\nodejs\npx.cmd" tsc --project tsconfig.json

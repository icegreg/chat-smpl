$env:PATH = "C:\Program Files\nodejs;" + $env:PATH
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
& "C:\Program Files\nodejs\npx.cmd" tsc --project e2e-selenium/tsconfig.json

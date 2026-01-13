$env:HEADLESS = "true"
$env:BASE_URL = "http://127.0.0.1:8888"
$env:PATH = "C:\Program Files\nodejs;" + $env:PATH
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
$env:TS_NODE_TRANSPILE_ONLY = "true"
& "C:\Program Files\nodejs\node.exe" node_modules/mocha/bin/mocha --require ts-node/register --timeout 120000 "e2e-selenium/tests/voice-audio-quality.spec.ts"

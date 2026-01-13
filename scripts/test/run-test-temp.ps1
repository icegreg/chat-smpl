$ErrorActionPreference = "Continue"
$env:PATH = "C:\Program Files\nodejs;" + $env:PATH
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"
$env:HEADLESS = "false"
$env:BASE_URL = "http://127.0.0.1:8888"

Write-Host "Starting voice audio quality test..."
& "C:\Program Files\nodejs\node.exe" "--loader=ts-node/esm" "--no-warnings" "./node_modules/mocha/bin/mocha.js" "--timeout" "300000" "--slow" "30000" "--no-config" "--extension=ts" "e2e-selenium/tests/voice-audio-quality.spec.ts"
$exitCode = $LASTEXITCODE
Write-Host "Test finished with exit code: $exitCode"
exit $exitCode

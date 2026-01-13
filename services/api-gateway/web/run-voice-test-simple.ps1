$env:BASE_URL = "http://127.0.0.1:8888"
$env:HEADLESS = "true"

Set-Location $PSScriptRoot

Write-Host "Running voice call test..."

# Run with --no-config to skip .mocharc.json (which uses ts-node for .ts files)
& "C:\Program Files\nodejs\node.exe" node_modules/mocha/bin/mocha.js `
    --no-config `
    --timeout 120000 `
    --reporter spec `
    "e2e-selenium/dist/tests/voice-call.spec.js"

Write-Host "Exit code: $LASTEXITCODE"

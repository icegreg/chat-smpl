# Build script
$env:PATH = "C:\Program Files\nodejs;$env:PATH"
Set-Location "C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web"

Write-Host "Building Vue app..."
npm run build
Write-Host "Build completed with exit code: $LASTEXITCODE"
exit $LASTEXITCODE

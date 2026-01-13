# Generate protobuf Go code from proto files
$ErrorActionPreference = "Continue"

$projectRoot = Split-Path -Parent $PSScriptRoot

Write-Host "Generating protobuf code for voice service..." -ForegroundColor Cyan
Write-Host "Project root: $projectRoot"

# Use Docker with namely/protoc-all which includes all necessary plugins
docker run --rm `
    -v "${projectRoot}:/defs" `
    namely/protoc-all:1.51_2 `
    -d proto/voice -l go -o proto/voice --go-source-relative

if ($LASTEXITCODE -eq 0) {
    Write-Host "Protobuf generation completed successfully!" -ForegroundColor Green
} else {
    Write-Host "Protobuf generation failed with exit code $LASTEXITCODE" -ForegroundColor Red
}

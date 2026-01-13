# Setup rtuccli alias for Windows PowerShell

$ProfilePath = $PROFILE.CurrentUserAllHosts
$AliasFunction = @'

# rtuccli - chat-smpl CLI tool
function rtuccli {
    param(
        [Parameter(ValueFromRemainingArguments=$true)]
        [string[]]$Arguments
    )
    $cmd = $Arguments -join ' '
    docker-compose exec -T admin-service sh -c $cmd
}
'@

# Create profile directory if it doesn't exist
$ProfileDir = Split-Path -Parent $ProfilePath
if (!(Test-Path $ProfileDir)) {
    New-Item -ItemType Directory -Path $ProfileDir -Force | Out-Null
    Write-Host "Created PowerShell profile directory: $ProfileDir" -ForegroundColor Green
}

# Create profile file if it doesn't exist
if (!(Test-Path $ProfilePath)) {
    New-Item -ItemType File -Path $ProfilePath -Force | Out-Null
    Write-Host "Created PowerShell profile: $ProfilePath" -ForegroundColor Green
}

# Check if alias already exists
$ProfileContent = Get-Content $ProfilePath -Raw -ErrorAction SilentlyContinue
if ($ProfileContent -match "function rtuccli") {
    Write-Host "Function 'rtuccli' already exists in PowerShell profile" -ForegroundColor Green
} else {
    Write-Host "Adding rtuccli function to PowerShell profile..." -ForegroundColor Yellow
    Add-Content -Path $ProfilePath -Value $AliasFunction
    Write-Host "Function added to $ProfilePath" -ForegroundColor Green
}

Write-Host ""
Write-Host "Setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "To use the function immediately, run:" -ForegroundColor Cyan
Write-Host '  . $PROFILE' -ForegroundColor White
Write-Host ""
Write-Host "Or open a new PowerShell window." -ForegroundColor Cyan
Write-Host ""
Write-Host "Usage examples:" -ForegroundColor Cyan
Write-Host "  rtuccli service list" -ForegroundColor White
Write-Host "  rtuccli conf list" -ForegroundColor White
Write-Host ""

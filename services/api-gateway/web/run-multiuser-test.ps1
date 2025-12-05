param(
    [int]$UserCount = 10,
    [int]$Duration = 5,
    [switch]$Headed
)

$env:Path = "C:\Program Files\nodejs;$env:Path"
Set-Location 'C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web'

$env:USER_COUNT = $UserCount
$env:TEST_DURATION = $Duration
$env:BASE_URL = 'http://127.0.0.1:8888'

if ($Headed) {
    $env:HEADLESS = 'false'
} else {
    $env:HEADLESS = 'true'
}

Write-Host "=== Multi-User Chat E2E Test ===" -ForegroundColor Cyan
Write-Host "Users: $UserCount"
Write-Host "Duration: $Duration minutes"
Write-Host "Headless: $(-not $Headed)"
Write-Host "Base URL: $env:BASE_URL"
Write-Host ""

& "C:\Program Files\nodejs\npm.cmd" run test:e2e:multiuser

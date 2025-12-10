# Load Test Runner
# Запуск нагрузочного тестирования с визуализацией в Grafana
param(
    [ValidateSet('smoke', 'load', 'stress', 'spike', 'soak')]
    [string]$Scenario = 'smoke',

    [ValidateSet('api', 'websocket', 'combined')]
    [string]$Type = 'api',

    [string]$BaseUrl = 'http://localhost:8888',

    [switch]$StartInfra,
    [switch]$StopInfra
)

$ErrorActionPreference = "Continue"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Chat App Load Testing" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Запуск инфраструктуры для метрик
if ($StartInfra) {
    Write-Host "Starting monitoring infrastructure..." -ForegroundColor Yellow
    Set-Location $scriptDir
    docker compose up -d influxdb grafana
    Start-Sleep -Seconds 5
    Write-Host "Grafana available at: http://localhost:3001 (admin/admin)" -ForegroundColor Green
    Write-Host ""
}

if ($StopInfra) {
    Write-Host "Stopping monitoring infrastructure..." -ForegroundColor Yellow
    Set-Location $scriptDir
    docker compose down
    exit 0
}

# Проверка доступности сервера
Write-Host "Checking server availability at $BaseUrl..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri $BaseUrl -TimeoutSec 5 -UseBasicParsing
    Write-Host "  Server is available (HTTP $($response.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "  WARNING: Server may not be available at $BaseUrl" -ForegroundColor Red
    Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Make sure the chat application is running:" -ForegroundColor Yellow
    Write-Host "  cd .. && docker compose up -d" -ForegroundColor Cyan
    exit 1
}
Write-Host ""

# Определяем тестовый скрипт
$testScript = switch ($Type) {
    'api' { 'api-load-test.js' }
    'websocket' { 'websocket-load-test.js' }
    'combined' { 'combined-load-test.js' }
}

# Параметры k6 в зависимости от сценария
$k6Params = switch ($Scenario) {
    'smoke' { '--vus 5 --duration 30s' }
    'load' { '--vus 50 --duration 5m' }
    'stress' { '--vus 100 --duration 10m' }
    'spike' { '--vus 200 --duration 3m' }
    'soak' { '--vus 50 --duration 30m' }
}

Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  Scenario: $Scenario"
Write-Host "  Type: $Type"
Write-Host "  Script: $testScript"
Write-Host "  K6 params: $k6Params"
Write-Host ""

# Проверяем наличие k6
$k6Installed = $null
try {
    $k6Installed = Get-Command k6 -ErrorAction SilentlyContinue
} catch {}

if ($k6Installed) {
    Write-Host "Running k6 locally..." -ForegroundColor Yellow

    # Запускаем k6 с отправкой метрик в InfluxDB (если инфра запущена)
    $influxRunning = docker ps --filter "name=loadtest-influxdb" --format "{{.Names}}" 2>$null
    if ($influxRunning) {
        Write-Host "  Sending metrics to InfluxDB" -ForegroundColor Cyan
        $env:K6_OUT = "influxdb=http://localhost:8086/k6"
    }

    $env:BASE_URL = $BaseUrl
    $env:SCENARIO = $Scenario

    Set-Location $scriptDir
    Invoke-Expression "k6 run $k6Params $testScript"
} else {
    Write-Host "k6 not found locally, using Docker..." -ForegroundColor Yellow

    Set-Location $scriptDir
    docker compose run --rm `
        -e BASE_URL=$BaseUrl `
        -e SCENARIO=$Scenario `
        k6 run $k6Params /scripts/$testScript
}

$exitCode = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($exitCode -eq 0) {
    Write-Host "  Load Test PASSED" -ForegroundColor Green
} else {
    Write-Host "  Load Test FAILED (exit code: $exitCode)" -ForegroundColor Red
}
Write-Host "========================================" -ForegroundColor Cyan

if ($influxRunning) {
    Write-Host ""
    Write-Host "View results in Grafana: http://localhost:3001" -ForegroundColor Cyan
}

exit $exitCode

# Network Resilience E2E Test Runner
# Тестирование устойчивости клиента к сетевым проблемам

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Network Resilience E2E Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Переходим в директорию web
$webDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $webDir

# Проверяем переменные окружения
$env:BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://127.0.0.1:8888" }
$env:HEADLESS = if ($env:HEADLESS) { $env:HEADLESS } else { "true" }

Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  BASE_URL: $env:BASE_URL"
Write-Host "  HEADLESS: $env:HEADLESS"
Write-Host ""

# Проверяем что сервер доступен
Write-Host "Checking server availability..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri $env:BASE_URL -TimeoutSec 5 -UseBasicParsing
    Write-Host "  Server is available (HTTP $($response.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "  WARNING: Server may not be available at $env:BASE_URL" -ForegroundColor Red
    Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

# Добавляем Node.js в PATH
$nodePath = "C:\Program Files\nodejs"
if (Test-Path $nodePath) {
    $env:PATH = "$nodePath;$env:PATH"
}

# Находим npm
$npmCmd = "$nodePath\npm.cmd"
if (-not (Test-Path $npmCmd)) {
    $npmCmd = "npm"  # Fallback to PATH
}
Write-Host "Using npm: $npmCmd" -ForegroundColor Yellow

# Запускаем тесты
Write-Host "Running network resilience tests..." -ForegroundColor Yellow
Write-Host ""

# Можно запустить все тесты или конкретный
$testFilter = $args[0]
if ($testFilter) {
    Write-Host "Filter: $testFilter" -ForegroundColor Cyan
    & $npmCmd run test:e2e -- --grep "$testFilter"
} else {
    # По умолчанию запускаем все network resilience тесты
    & $npmCmd run test:e2e -- --grep "Network Resilience"
}

$exitCode = $LASTEXITCODE

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($exitCode -eq 0) {
    Write-Host "  Tests PASSED" -ForegroundColor Green
} else {
    Write-Host "  Tests FAILED (exit code: $exitCode)" -ForegroundColor Red
}
Write-Host "========================================" -ForegroundColor Cyan

exit $exitCode

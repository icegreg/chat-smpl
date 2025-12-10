# Start Monitoring Stack
# Usage: .\start-monitoring.ps1

param(
    [switch]$Stop,
    [switch]$Logs
)

$ErrorActionPreference = "Continue"
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$rootDir = (Get-Item $scriptDir).Parent.Parent.FullName

Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host "  Chat Application Monitoring" -ForegroundColor Cyan
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host ""

if ($Stop) {
    Write-Host "Stopping monitoring stack..." -ForegroundColor Yellow
    Set-Location $rootDir
    docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml down
    Write-Host "Monitoring stopped" -ForegroundColor Green
    exit 0
}

if ($Logs) {
    Write-Host "Showing logs..." -ForegroundColor Yellow
    Set-Location $rootDir
    docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml logs -f
    exit 0
}

# Check if main services are running
Write-Host "Checking main services..." -ForegroundColor Yellow
$containers = docker ps --format "{{.Names}}" 2>$null
if (-not ($containers -match "chatapp-")) {
    Write-Host "  WARNING: Main chat services are not running!" -ForegroundColor Red
    Write-Host "  Start them first with: docker compose up -d" -ForegroundColor Yellow
    Write-Host ""
}

# Create network if not exists
docker network inspect chatapp-network 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Creating network chatapp-network..." -ForegroundColor Yellow
    docker network create chatapp-network
}

# Enable RabbitMQ Prometheus plugin
Write-Host "Enabling RabbitMQ Prometheus plugin..." -ForegroundColor Yellow
docker exec chatapp-rabbitmq rabbitmq-plugins enable rabbitmq_prometheus 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "  RabbitMQ Prometheus plugin enabled" -ForegroundColor Green
} else {
    Write-Host "  Note: RabbitMQ may not be running yet" -ForegroundColor Yellow
}

# Start monitoring stack
Write-Host ""
Write-Host "Starting monitoring stack..." -ForegroundColor Yellow
Set-Location $rootDir
docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml up -d prometheus grafana redis-exporter postgres-exporter

Write-Host ""
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host "  Monitoring Started!" -ForegroundColor Green
Write-Host "=" * 60 -ForegroundColor Cyan
Write-Host ""
Write-Host "Access URLs:" -ForegroundColor Yellow
Write-Host "  Grafana:       http://localhost:3000  (admin/admin)" -ForegroundColor White
Write-Host "  Prometheus:    http://localhost:9090" -ForegroundColor White
Write-Host "  RabbitMQ:      http://localhost:15672 (chatapp/secret)" -ForegroundColor White
Write-Host "  Centrifugo:    http://localhost:8000  (admin)" -ForegroundColor White
Write-Host ""
Write-Host "Grafana Dashboards:" -ForegroundColor Yellow
Write-Host "  - Chat Services Overview" -ForegroundColor White
Write-Host "  - Container Resources" -ForegroundColor White
Write-Host ""
Write-Host "To stop monitoring: .\start-monitoring.ps1 -Stop" -ForegroundColor Gray
Write-Host "To view logs:       .\start-monitoring.ps1 -Logs" -ForegroundColor Gray
Write-Host ""

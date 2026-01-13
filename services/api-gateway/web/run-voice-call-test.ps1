# Voice Call E2E Test Runner
# Runs WebRTC voice call tests with non-headless Chrome
#
# Usage:
#   .\run-voice-call-test.ps1                    # Use localhost
#   .\run-voice-call-test.ps1 -External          # Use external IP (10.99.22.46)
#   .\run-voice-call-test.ps1 -BaseUrl "http://custom:8888"
#   .\run-voice-call-test.ps1 -Headless          # Run headless (not recommended for WebRTC)

param(
    [switch]$Headless,
    [switch]$External,
    [string]$BaseUrl
)

# Set BaseUrl based on flags
# External access uses HTTPS (port 8443) because getUserMedia requires secure context
if ($External) {
    $BaseUrl = "https://10.99.22.46:8443"
} elseif (-not $BaseUrl) {
    $BaseUrl = "http://127.0.0.1:8888"
}

$ErrorActionPreference = "Stop"

# Find node executable
function Get-NodePath {
    # Try common locations
    $nodePaths = @(
        "C:\Program Files\nodejs\node.exe",
        "C:\Program Files (x86)\nodejs\node.exe",
        "$env:APPDATA\nvm\v*\node.exe",
        "$env:LOCALAPPDATA\Programs\nodejs\node.exe"
    )

    foreach ($path in $nodePaths) {
        $resolved = Get-Item -Path $path -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($resolved) {
            return $resolved.FullName
        }
    }

    # Try finding in PATH via cmd.exe (which has different PATH)
    $cmdResult = cmd.exe /c "where node 2>nul" | Select-Object -First 1
    if ($cmdResult -and (Test-Path $cmdResult)) {
        return $cmdResult
    }

    # Check if node works directly
    try {
        $null = & node --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            return "node"
        }
    } catch {}

    return $null
}

$nodePath = Get-NodePath
if (-not $nodePath) {
    Write-Host "ERROR: Node.js not found. Please install Node.js or add it to PATH." -ForegroundColor Red
    exit 1
}
Write-Host "Using Node: $nodePath" -ForegroundColor Gray

# Set environment variables
$env:BASE_URL = $BaseUrl

# By default, run in non-headless mode for WebRTC
if ($Headless) {
    $env:HEADLESS = "true"
    Write-Host "Running in HEADLESS mode (WebRTC may have issues)" -ForegroundColor Yellow
} else {
    $env:HEADLESS = "false"
    Write-Host "Running in NON-HEADLESS mode (recommended for WebRTC)" -ForegroundColor Green
}

Write-Host ""
Write-Host "=== Voice Call E2E Test ===" -ForegroundColor Cyan
Write-Host "Base URL: $BaseUrl"
Write-Host "Headless: $($env:HEADLESS)"
Write-Host ""

# Check if FreeSWITCH is running
Write-Host "Checking FreeSWITCH status..." -ForegroundColor Yellow
$fsStatus = docker ps --filter "name=chatapp-freeswitch" --format "{{.Status}}" 2>$null
if ($fsStatus) {
    Write-Host "FreeSWITCH is running: $fsStatus" -ForegroundColor Green
} else {
    Write-Host "WARNING: FreeSWITCH container not found or not running" -ForegroundColor Red
    Write-Host "WebRTC connection tests may fail" -ForegroundColor Yellow
}

Write-Host ""

# Change to web directory
Push-Location $PSScriptRoot

try {
    # Compile TypeScript
    Write-Host "Compiling TypeScript..." -ForegroundColor Yellow
    & $nodePath node_modules/typescript/bin/tsc --project e2e-selenium/tsconfig.json
    if ($LASTEXITCODE -ne 0) {
        Write-Host "TypeScript compilation failed" -ForegroundColor Red
        exit 1
    }
    Write-Host "TypeScript compiled successfully" -ForegroundColor Green

    # Run the voice call tests
    Write-Host ""
    Write-Host "Running Voice Call tests..." -ForegroundColor Cyan
    Write-Host ""

    # Use --no-config to skip .mocharc.json (which uses ts-node for .ts files)
    & $nodePath node_modules/mocha/bin/mocha.js `
        --no-config `
        --timeout 120000 `
        --reporter spec `
        e2e-selenium/dist/tests/voice-call.spec.js

    $testResult = $LASTEXITCODE

    Write-Host ""
    if ($testResult -eq 0) {
        Write-Host "=== ALL TESTS PASSED ===" -ForegroundColor Green
    } else {
        Write-Host "=== SOME TESTS FAILED ===" -ForegroundColor Red
    }

    exit $testResult
}
finally {
    Pop-Location
}

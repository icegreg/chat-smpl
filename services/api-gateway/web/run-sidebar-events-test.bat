@echo off
setlocal

set HEADLESS=false
set BASE_URL=http://127.0.0.1:8888
set PATH=C:\Program Files\nodejs;%PATH%

cd /d "%~dp0"

echo === Running Sidebar Events E2E Test ===
echo HEADLESS: %HEADLESS%
echo BASE_URL: %BASE_URL%
echo.

call npx mocha --require ts-node/register e2e-selenium/tests/sidebar-events.spec.ts --timeout 120000

echo.
echo === Test Complete ===

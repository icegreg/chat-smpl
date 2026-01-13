@echo off
REM Setup rtuccli alias for Windows CMD (doskey macros)
REM Note: This needs to be run every time you start CMD

echo Setting up rtuccli alias for CMD...
echo.

doskey rtuccli=docker-compose exec -T admin-service sh -c $*

echo.
echo ✓ Alias 'rtuccli' has been set up for this CMD session!
echo.
echo IMPORTANT: CMD doskey macros only work in the current session.
echo To make this permanent, you need to:
echo   1. Create a batch file with the doskey command
echo   2. Set it to run automatically via registry
echo.
echo Or use PowerShell instead (recommended):
echo   Run: powershell -ExecutionPolicy Bypass -File scripts\setup-rtuccli-alias.ps1
echo.
echo Usage examples:
echo   rtuccli service list
echo   rtuccli conf list --status active
echo.
pause

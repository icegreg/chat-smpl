@echo off
cd /d "%~dp0"
set PATH=C:\Program Files\nodejs;%PATH%
call npx vue-tsc --noEmit

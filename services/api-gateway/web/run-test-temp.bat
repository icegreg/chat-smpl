@echo off
cd /d C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web
set HEADLESS=false
set BASE_URL=http://127.0.0.1:8888
set PATH=C:\Program Files\nodejs;%PATH%
call npx mocha --require ts-node/register e2e-selenium/tests/sidebar-events.spec.ts --timeout 180000 2>&1

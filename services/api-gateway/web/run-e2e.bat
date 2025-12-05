@echo off
cd /d C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web
set HEADLESS=false
set BASE_URL=http://127.0.0.1:8888
echo Starting E2E tests...
call npm run test:e2e -- --grep "Messaging" > e2e-output.txt 2>&1
type e2e-output.txt

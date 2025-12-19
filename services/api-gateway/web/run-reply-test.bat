@echo off
cd /d C:\Users\ryabikov\dev\testing\chat-smpl\services\api-gateway\web
set HEADLESS=false
set BASE_URL=http://127.0.0.1:8888
"C:\Program Files\nodejs\npx.cmd" mocha --require ts-node/register e2e-selenium/tests/reply-forward.spec.ts --timeout 120000

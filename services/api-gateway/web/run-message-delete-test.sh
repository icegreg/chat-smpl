#!/bin/bash

# Message Delete & Restore E2E Test
# Run in visible browser mode (not headless)

cd "$(dirname "$0")"

export HEADLESS=false
export BASE_URL=http://127.0.0.1:8888

echo "=== Message Delete & Restore E2E Test ==="
echo "Running in visible browser mode (HEADLESS=false)"
echo ""

npm run test:e2e -- --grep "Message Delete"

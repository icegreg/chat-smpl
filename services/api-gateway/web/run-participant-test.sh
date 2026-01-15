#!/bin/bash

# Run Participant Management E2E Tests
# Usage: ./run-participant-test.sh

set -e

echo "=== Running Participant Management E2E Tests ==="

# Set environment variables
export BASE_URL=${BASE_URL:-"http://127.0.0.1:8888"}
export HEADLESS=${HEADLESS:-"true"}

echo "BASE_URL: $BASE_URL"
echo "HEADLESS: $HEADLESS"

# Change to script directory
cd "$(dirname "$0")"

# Run the test
echo ""
echo "Running participant management tests..."
npx mocha --require ts-node/register --timeout 120000 "e2e-selenium/tests/participant-management.spec.ts"

if [ $? -eq 0 ]; then
    echo ""
    echo "=== Tests Passed ==="
else
    echo ""
    echo "=== Tests Failed ==="
    exit 1
fi

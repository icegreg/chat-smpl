#!/bin/bash
# Run API User Simulation Tests with timing measurements
# Usage: ./run-api-tests.sh [scenario|multiuser|all]

set -e
cd "$(dirname "$0")"

TEST_TYPE="${1:-all}"
API_URL="${API_URL:-http://127.0.0.1:8888}"

echo ""
echo "========================================"
echo "  API User Simulation Tests"
echo "========================================"
echo ""
echo "API URL: $API_URL"
echo "Test Type: $TEST_TYPE"
echo ""

export API_URL

# Check API availability
echo "Checking API availability..."
if curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/auth/me" | grep -q "401"; then
    echo "API is available (got 401 - expected)"
else
    echo "Warning: API may not be available at $API_URL"
fi
echo ""

# Define test files
SCENARIO_TESTS="e2e-selenium/tests/api-user-scenarios.spec.ts"
MULTIUSER_TESTS="e2e-selenium/tests/api-multiuser-simulation.spec.ts"

case "$TEST_TYPE" in
    scenario)
        TEST_FILES="$SCENARIO_TESTS"
        echo "Running: User Scenario Tests"
        ;;
    multiuser)
        TEST_FILES="$MULTIUSER_TESTS"
        echo "Running: Multi-User Simulation Tests"
        ;;
    all)
        TEST_FILES="$SCENARIO_TESTS $MULTIUSER_TESTS"
        echo "Running: All API Tests"
        ;;
    *)
        echo "Unknown test type: $TEST_TYPE"
        echo "Available options: scenario, multiuser, all"
        exit 1
        ;;
esac

echo ""
echo "Starting tests..."
echo "----------------------------------------"
echo ""

npx mocha --require ts-node/register --timeout 300000 $TEST_FILES

echo ""
echo "----------------------------------------"
echo "Tests completed"

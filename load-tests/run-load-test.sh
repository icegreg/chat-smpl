#!/bin/bash
# Load Test Runner
# Запуск нагрузочного тестирования с визуализацией в Grafana
# Usage: ./run-load-test.sh [OPTIONS]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default values
SCENARIO="smoke"
TYPE="api"
BASE_URL="http://localhost:8888"
START_INFRA=false
STOP_INFRA=false

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -s, --scenario SCENARIO   Test scenario: smoke, load, stress, spike, soak (default: smoke)"
    echo "  -t, --type TYPE           Test type: api, websocket, combined (default: api)"
    echo "  -u, --url URL             Base URL (default: http://localhost:8888)"
    echo "  --start-infra             Start monitoring infrastructure (InfluxDB + Grafana)"
    echo "  --stop-infra              Stop monitoring infrastructure"
    echo "  -h, --help                Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 --scenario smoke --type api"
    echo "  $0 -s load -t websocket"
    echo "  $0 --start-infra"
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--scenario)
            SCENARIO="$2"
            shift 2
            ;;
        -t|--type)
            TYPE="$2"
            shift 2
            ;;
        -u|--url)
            BASE_URL="$2"
            shift 2
            ;;
        --start-infra)
            START_INFRA=true
            shift
            ;;
        --stop-infra)
            STOP_INFRA=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Validate scenario
case $SCENARIO in
    smoke|load|stress|spike|soak) ;;
    *)
        echo -e "${RED}Invalid scenario: $SCENARIO${NC}"
        echo "Valid scenarios: smoke, load, stress, spike, soak"
        exit 1
        ;;
esac

# Validate type
case $TYPE in
    api|websocket|combined) ;;
    *)
        echo -e "${RED}Invalid type: $TYPE${NC}"
        echo "Valid types: api, websocket, combined"
        exit 1
        ;;
esac

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Chat App Load Testing${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

# Start infrastructure
if [ "$START_INFRA" = true ]; then
    echo -e "${YELLOW}Starting monitoring infrastructure...${NC}"
    cd "$SCRIPT_DIR"
    docker compose up -d influxdb grafana
    sleep 5
    echo -e "${GREEN}Grafana available at: http://localhost:3001 (admin/admin)${NC}"
    echo ""
fi

# Stop infrastructure
if [ "$STOP_INFRA" = true ]; then
    echo -e "${YELLOW}Stopping monitoring infrastructure...${NC}"
    cd "$SCRIPT_DIR"
    docker compose down
    exit 0
fi

# Check server availability
echo -e "${YELLOW}Checking server availability at $BASE_URL...${NC}"
if curl -s --connect-timeout 5 "$BASE_URL" > /dev/null 2>&1; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL" 2>/dev/null || echo "000")
    echo -e "  ${GREEN}Server is available (HTTP $HTTP_CODE)${NC}"
else
    echo -e "  ${RED}WARNING: Server may not be available at $BASE_URL${NC}"
    echo ""
    echo -e "${YELLOW}Make sure the chat application is running:${NC}"
    echo -e "${CYAN}  cd .. && docker compose up -d${NC}"
    exit 1
fi
echo ""

# Determine test script
case $TYPE in
    api) TEST_SCRIPT="api-load-test.js" ;;
    websocket) TEST_SCRIPT="websocket-load-test.js" ;;
    combined) TEST_SCRIPT="combined-load-test.js" ;;
esac

# K6 parameters based on scenario
case $SCENARIO in
    smoke) K6_PARAMS="--vus 5 --duration 30s" ;;
    load) K6_PARAMS="--vus 50 --duration 5m" ;;
    stress) K6_PARAMS="--vus 100 --duration 10m" ;;
    spike) K6_PARAMS="--vus 200 --duration 3m" ;;
    soak) K6_PARAMS="--vus 50 --duration 30m" ;;
esac

echo -e "${YELLOW}Configuration:${NC}"
echo "  Scenario: $SCENARIO"
echo "  Type: $TYPE"
echo "  Script: $TEST_SCRIPT"
echo "  K6 params: $K6_PARAMS"
echo ""

# Check if k6 is installed
INFLUX_RUNNING=""
if command -v k6 &> /dev/null; then
    echo -e "${YELLOW}Running k6 locally...${NC}"

    # Check if InfluxDB is running for metrics
    INFLUX_RUNNING=$(docker ps --filter "name=loadtest-influxdb" --format "{{.Names}}" 2>/dev/null || true)
    if [ -n "$INFLUX_RUNNING" ]; then
        echo -e "  ${CYAN}Sending metrics to InfluxDB${NC}"
        export K6_OUT="influxdb=http://localhost:8086/k6"
    fi

    export BASE_URL
    export SCENARIO

    cd "$SCRIPT_DIR"
    # shellcheck disable=SC2086
    k6 run $K6_PARAMS "$TEST_SCRIPT"
    EXIT_CODE=$?
else
    echo -e "${YELLOW}k6 not found locally, using Docker...${NC}"

    cd "$SCRIPT_DIR"
    # shellcheck disable=SC2086
    docker compose run --rm \
        -e BASE_URL="$BASE_URL" \
        -e SCENARIO="$SCENARIO" \
        k6 run $K6_PARAMS "/scripts/$TEST_SCRIPT"
    EXIT_CODE=$?
fi

echo ""
echo -e "${CYAN}========================================${NC}"
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}  Load Test PASSED${NC}"
else
    echo -e "${RED}  Load Test FAILED (exit code: $EXIT_CODE)${NC}"
fi
echo -e "${CYAN}========================================${NC}"

if [ -n "$INFLUX_RUNNING" ]; then
    echo ""
    echo -e "${CYAN}View results in Grafana: http://localhost:3001${NC}"
fi

exit $EXIT_CODE

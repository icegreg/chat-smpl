#!/bin/bash
# Extreme Load Test Orchestrator
# Запуск полного нагрузочного теста:
# - 400 k6 VU генерируют 10 msg/sec
# - 10 браузеров с медленной сетью
# - 20 браузеров с обрывающейся сетью
# - 20% сообщений с файлами (UUID внутри)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$SCRIPT_DIR/../services/api-gateway/web"

# Default values
K6_USERS=400
TARGET_MPS=10
SLOW_BROWSERS=10
FLAKY_BROWSERS=20
DURATION_SECONDS=300
FILE_RATIO=0.2
BASE_URL="http://127.0.0.1:8888"
K6_ONLY=false
BROWSERS_ONLY=false
DRY_RUN=false

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
    echo "  --k6-users NUM         Number of k6 virtual users (default: 400)"
    echo "  --target-mps NUM       Target messages per second (default: 10)"
    echo "  --slow-browsers NUM    Number of slow browser clients (default: 10)"
    echo "  --flaky-browsers NUM   Number of flaky browser clients (default: 20)"
    echo "  --duration NUM         Test duration in seconds (default: 300)"
    echo "  --file-ratio NUM       Ratio of messages with files 0.0-1.0 (default: 0.2)"
    echo "  --url URL              Base URL (default: http://127.0.0.1:8888)"
    echo "  --k6-only              Run only k6 tests (no browsers)"
    echo "  --browsers-only        Run only browser tests (no k6)"
    echo "  --dry-run              Show configuration without running tests"
    echo "  -h, --help             Show this help"
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --k6-users)
            K6_USERS="$2"
            shift 2
            ;;
        --target-mps)
            TARGET_MPS="$2"
            shift 2
            ;;
        --slow-browsers)
            SLOW_BROWSERS="$2"
            shift 2
            ;;
        --flaky-browsers)
            FLAKY_BROWSERS="$2"
            shift 2
            ;;
        --duration)
            DURATION_SECONDS="$2"
            shift 2
            ;;
        --file-ratio)
            FILE_RATIO="$2"
            shift 2
            ;;
        --url)
            BASE_URL="$2"
            shift 2
            ;;
        --k6-only)
            K6_ONLY=true
            shift
            ;;
        --browsers-only)
            BROWSERS_ONLY=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
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

print_separator() {
    echo -e "${CYAN}======================================================================${NC}"
}

print_separator
echo -e "${CYAN}  EXTREME LOAD TEST${NC}"
print_separator
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Base URL:        $BASE_URL"
echo "  k6 Users:        $K6_USERS"
echo "  Target MPS:      $TARGET_MPS msg/sec"
echo "  Slow Browsers:   $SLOW_BROWSERS"
echo "  Flaky Browsers:  $FLAKY_BROWSERS"
echo "  Duration:        $DURATION_SECONDS seconds"
echo "  File Ratio:      $(echo "$FILE_RATIO * 100" | bc)%"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}DRY RUN - Not actually starting tests${NC}"
    exit 0
fi

# Check server availability
echo -e "${YELLOW}Checking server availability...${NC}"
if curl -s --connect-timeout 10 "$BASE_URL" > /dev/null 2>&1; then
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL" 2>/dev/null || echo "000")
    echo -e "  ${GREEN}Server is available (HTTP $HTTP_CODE)${NC}"
else
    echo -e "  ${RED}ERROR: Server not available at $BASE_URL${NC}"
    echo -e "  ${RED}Make sure docker compose up is running${NC}"
    exit 1
fi
echo ""

# Step 1: Create shared chat and get tokens
echo -e "${CYAN}Step 1: Creating shared chat...${NC}"

OWNER_USERNAME="extreme_owner_$RANDOM"
OWNER_EMAIL="${OWNER_USERNAME}@extreme.local"

# Register owner
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\": \"$OWNER_USERNAME\", \"email\": \"$OWNER_EMAIL\", \"password\": \"TestPass123!\"}")

OWNER_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.access_token // empty')

if [ -z "$OWNER_TOKEN" ]; then
    echo -e "  ${RED}Failed to register owner${NC}"
    echo "  Response: $REGISTER_RESPONSE"
    exit 1
fi
echo -e "  ${GREEN}Owner registered: $OWNER_USERNAME${NC}"

# Get owner ID
ME_RESPONSE=$(curl -s -X GET "$BASE_URL/api/auth/me" \
    -H "Authorization: Bearer $OWNER_TOKEN")

OWNER_ID=$(echo "$ME_RESPONSE" | jq -r '.id // empty')

if [ -z "$OWNER_ID" ]; then
    echo -e "  ${RED}Failed to get owner info${NC}"
    exit 1
fi

# Create chat
CHAT_NAME="Extreme Load Test $(date '+%Y-%m-%d %H:%M')"
CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/chats" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $OWNER_TOKEN" \
    -d "{\"type\": \"group\", \"name\": \"$CHAT_NAME\", \"description\": \"$K6_USERS k6 users + $SLOW_BROWSERS slow browsers + $FLAKY_BROWSERS flaky browsers\", \"participant_ids\": []}")

CHAT_ID=$(echo "$CHAT_RESPONSE" | jq -r '.id // empty')

if [ -z "$CHAT_ID" ]; then
    echo -e "  ${RED}Failed to create chat${NC}"
    echo "  Response: $CHAT_RESPONSE"
    exit 1
fi
echo -e "  ${GREEN}Chat created: $CHAT_ID${NC}"
echo ""

# Export for child processes
export CHAT_ID
export OWNER_TOKEN
export BASE_URL

# Temporary files for job output
K6_OUTPUT_FILE=$(mktemp)
BROWSER_OUTPUT_FILE=$(mktemp)
K6_PID=""
BROWSER_PID=""

cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    [ -n "$K6_PID" ] && kill "$K6_PID" 2>/dev/null || true
    [ -n "$BROWSER_PID" ] && kill "$BROWSER_PID" 2>/dev/null || true
    rm -f "$K6_OUTPUT_FILE" "$BROWSER_OUTPUT_FILE"
}
trap cleanup EXIT

# Step 2: Start k6 load generator
if [ "$BROWSERS_ONLY" = false ]; then
    echo -e "${CYAN}Step 2: Starting k6 load generator ($K6_USERS VUs, $TARGET_MPS msg/sec)...${NC}"

    K6_SCRIPT="$SCRIPT_DIR/extreme-load-test.js"

    if command -v k6 &> /dev/null; then
        echo -e "  ${GREEN}Using local k6${NC}"

        (
            cd "$SCRIPT_DIR"
            BASE_URL="$BASE_URL" \
            VUS="$K6_USERS" \
            TARGET_MPS="$TARGET_MPS" \
            FILE_RATIO="$FILE_RATIO" \
            DURATION="${DURATION_SECONDS}s" \
            k6 run \
                --vus "$K6_USERS" \
                --duration "${DURATION_SECONDS}s" \
                -e BASE_URL="$BASE_URL" \
                -e VUS="$K6_USERS" \
                -e TARGET_MPS="$TARGET_MPS" \
                -e FILE_RATIO="$FILE_RATIO" \
                "$K6_SCRIPT" \
                > "$K6_OUTPUT_FILE" 2>&1
        ) &
        K6_PID=$!
        echo -e "  ${GREEN}k6 started (PID: $K6_PID)${NC}"
    else
        echo -e "  ${GREEN}Using Docker k6${NC}"

        (
            cd "$SCRIPT_DIR"
            docker compose run --rm \
                -e BASE_URL=http://127.0.0.1:8888 \
                -e WS_URL=ws://127.0.0.1:8000 \
                -e VUS="$K6_USERS" \
                -e TARGET_MPS="$TARGET_MPS" \
                -e FILE_RATIO="$FILE_RATIO" \
                -e DURATION="${DURATION_SECONDS}s" \
                k6 run \
                --vus "$K6_USERS" \
                --duration "${DURATION_SECONDS}s" \
                extreme-load-test.js \
                > "$K6_OUTPUT_FILE" 2>&1
        ) &
        K6_PID=$!
        echo -e "  ${GREEN}k6 Docker started (PID: $K6_PID)${NC}"
    fi
fi
echo ""

# Step 3: Start browser clients
if [ "$K6_ONLY" = false ]; then
    echo -e "${CYAN}Step 3: Starting browser clients ($SLOW_BROWSERS slow + $FLAKY_BROWSERS flaky)...${NC}"

    # Delay browser start to let k6 ramp up
    echo -e "  ${YELLOW}Waiting 10 seconds for k6 to ramp up...${NC}"
    sleep 10

    (
        cd "$WEB_DIR"
        CHAT_ID="$CHAT_ID" \
        OWNER_TOKEN="$OWNER_TOKEN" \
        BASE_URL="$BASE_URL" \
        SLOW_CLIENTS="$SLOW_BROWSERS" \
        FLAKY_CLIENTS="$FLAKY_BROWSERS" \
        TEST_DURATION="$DURATION_SECONDS" \
        HEADLESS="true" \
        npm run test:e2e -- --grep "Extreme Browser Clients" \
        > "$BROWSER_OUTPUT_FILE" 2>&1
    ) &
    BROWSER_PID=$!
    echo -e "  ${GREEN}Browser tests started (PID: $BROWSER_PID)${NC}"
fi
echo ""

# Step 4: Monitor progress
echo -e "${CYAN}Step 4: Monitoring test progress...${NC}"
echo -e "  ${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

START_TIME=$(date +%s)
END_TIME=$((START_TIME + DURATION_SECONDS + 60))  # Extra minute for cleanup

while [ "$(date +%s)" -lt "$END_TIME" ]; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    REMAINING=$((DURATION_SECONDS - ELAPSED))
    [ "$REMAINING" -lt 0 ] && REMAINING=0

    printf "\r  Elapsed: %ds | Remaining: %ds  " "$ELAPSED" "$REMAINING"

    # Check if k6 job is still running
    if [ -n "$K6_PID" ] && ! kill -0 "$K6_PID" 2>/dev/null; then
        echo ""
        echo -e "  ${GREEN}k6 job completed${NC}"
        K6_PID=""
    fi

    # Check if browser job is still running
    if [ -n "$BROWSER_PID" ] && ! kill -0 "$BROWSER_PID" 2>/dev/null; then
        echo ""
        echo -e "  ${GREEN}Browser job completed${NC}"
        BROWSER_PID=""
    fi

    # Exit if both jobs are done
    if [ -z "$K6_PID" ] && [ -z "$BROWSER_PID" ]; then
        break
    fi

    sleep 5
done

echo ""

# Step 5: Collect results
echo ""
echo -e "${CYAN}Step 5: Collecting results...${NC}"

if [ -f "$K6_OUTPUT_FILE" ] && [ -s "$K6_OUTPUT_FILE" ]; then
    echo ""
    echo -e "${YELLOW}k6 Results:${NC}"
    tail -50 "$K6_OUTPUT_FILE" | while read -r line; do
        echo "  $line"
    done
fi

if [ -f "$BROWSER_OUTPUT_FILE" ] && [ -s "$BROWSER_OUTPUT_FILE" ]; then
    echo ""
    echo -e "${YELLOW}Browser Results:${NC}"
    tail -30 "$BROWSER_OUTPUT_FILE" | while read -r line; do
        echo "  $line"
    done
fi

# Summary
echo ""
print_separator
echo -e "${CYAN}  TEST COMPLETED${NC}"
print_separator
echo ""
echo -e "${YELLOW}Chat ID: $CHAT_ID${NC}"
echo -e "${YELLOW}You can view the chat at: $BASE_URL/chat/$CHAT_ID${NC}"
echo ""

# Get message count from API
MESSAGES_RESPONSE=$(curl -s -X GET "$BASE_URL/api/chats/$CHAT_ID/messages?limit=1" \
    -H "Authorization: Bearer $OWNER_TOKEN" 2>/dev/null || echo "{}")

TOTAL_MESSAGES=$(echo "$MESSAGES_RESPONSE" | jq -r '.total // "unknown"')
echo -e "${GREEN}Total messages in chat: $TOTAL_MESSAGES${NC}"

echo ""
echo -e "${GREEN}Test completed!${NC}"

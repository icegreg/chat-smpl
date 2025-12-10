#!/bin/bash
# Start Monitoring Stack
# Usage: ./start-monitoring.sh [-s|--stop] [-l|--logs]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${CYAN}============================================================${NC}"
    echo -e "${CYAN}  Chat Application Monitoring${NC}"
    echo -e "${CYAN}============================================================${NC}"
    echo ""
}

# Parse arguments
STOP=false
LOGS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--stop)
            STOP=true
            shift
            ;;
        -l|--logs)
            LOGS=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [-s|--stop] [-l|--logs] [-h|--help]"
            echo ""
            echo "Options:"
            echo "  -s, --stop    Stop monitoring stack"
            echo "  -l, --logs    Show logs"
            echo "  -h, --help    Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

print_header

if [ "$STOP" = true ]; then
    echo -e "${YELLOW}Stopping monitoring stack...${NC}"
    cd "$ROOT_DIR"
    docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml down
    echo -e "${GREEN}Monitoring stopped${NC}"
    exit 0
fi

if [ "$LOGS" = true ]; then
    echo -e "${YELLOW}Showing logs...${NC}"
    cd "$ROOT_DIR"
    docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml logs -f
    exit 0
fi

# Check if main services are running
echo -e "${YELLOW}Checking main services...${NC}"
containers=$(docker ps --format "{{.Names}}" 2>/dev/null || true)
if ! echo "$containers" | grep -q "chatapp-"; then
    echo -e "  ${RED}WARNING: Main chat services are not running!${NC}"
    echo -e "  ${YELLOW}Start them first with: docker compose up -d${NC}"
    echo ""
fi

# Create network if not exists
if ! docker network inspect chatapp-network &>/dev/null; then
    echo -e "${YELLOW}Creating network chatapp-network...${NC}"
    docker network create chatapp-network
fi

# Enable RabbitMQ Prometheus plugin
echo -e "${YELLOW}Enabling RabbitMQ Prometheus plugin...${NC}"
if docker exec chatapp-rabbitmq rabbitmq-plugins enable rabbitmq_prometheus 2>/dev/null; then
    echo -e "  ${GREEN}RabbitMQ Prometheus plugin enabled${NC}"
else
    echo -e "  ${YELLOW}Note: RabbitMQ may not be running yet${NC}"
fi

# Start monitoring stack
echo ""
echo -e "${YELLOW}Starting monitoring stack...${NC}"
cd "$ROOT_DIR"
docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml up -d prometheus grafana redis-exporter postgres-exporter

echo ""
echo -e "${CYAN}============================================================${NC}"
echo -e "${GREEN}  Monitoring Started!${NC}"
echo -e "${CYAN}============================================================${NC}"
echo ""
echo -e "${YELLOW}Access URLs:${NC}"
echo -e "  ${WHITE}Grafana:       http://localhost:3000  (admin/admin)${NC}"
echo -e "  ${WHITE}Prometheus:    http://localhost:9090${NC}"
echo -e "  ${WHITE}RabbitMQ:      http://localhost:15672 (chatapp/secret)${NC}"
echo -e "  ${WHITE}Centrifugo:    http://localhost:8000  (admin)${NC}"
echo ""
echo -e "${YELLOW}Grafana Dashboards:${NC}"
echo -e "  ${WHITE}- Chat Services Overview${NC}"
echo -e "  ${WHITE}- Container Resources${NC}"
echo ""
echo -e "${GRAY}To stop monitoring: $0 --stop${NC}"
echo -e "${GRAY}To view logs:       $0 --logs${NC}"
echo ""

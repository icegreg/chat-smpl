#!/bin/bash
# Update Prometheus targets dynamically based on running containers
# Usage: ./update-targets.sh [service_name]
# Run this after scaling services: docker compose up -d --scale centrifugo=5

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGETS_DIR="$SCRIPT_DIR/targets"

mkdir -p "$TARGETS_DIR"

update_centrifugo_targets() {
    local containers=($(docker ps --format "{{.Names}}" | grep centrifugo | sort))
    local count=${#containers[@]}

    if [ $count -eq 0 ]; then
        echo "No centrifugo containers found"
        return
    fi

    # Build JSON array
    local json='[\n  {\n    "targets": ['
    local first=true
    for container in "${containers[@]}"; do
        if [ "$first" = true ]; then
            first=false
            json="$json\n      \"$container:8000\""
        else
            json="$json,\n      \"$container:8000\""
        fi
    done
    json="$json\n    ],\n    \"labels\": {\n      \"job\": \"centrifugo\"\n    }\n  }\n]"

    echo -e "$json" > "$TARGETS_DIR/centrifugo.json"
    echo "Updated centrifugo targets: $count instances"
}

update_users_service_targets() {
    local containers=($(docker ps --format "{{.Names}}" | grep users-service | sort))
    local count=${#containers[@]}

    if [ $count -eq 0 ]; then
        echo "No users-service containers found"
        return
    fi

    # Build JSON array
    local json='[\n  {\n    "targets": ['
    local first=true
    for container in "${containers[@]}"; do
        if [ "$first" = true ]; then
            first=false
            json="$json\n      \"$container:8081\""
        else
            json="$json,\n      \"$container:8081\""
        fi
    done
    json="$json\n    ],\n    \"labels\": {\n      \"job\": \"users-service\"\n    }\n  }\n]"

    echo -e "$json" > "$TARGETS_DIR/users-service.json"
    echo "Updated users-service targets: $count instances"
}

# Update all or specific service
case "${1:-all}" in
    centrifugo)
        update_centrifugo_targets
        ;;
    users-service)
        update_users_service_targets
        ;;
    all)
        update_centrifugo_targets
        update_users_service_targets
        ;;
    *)
        echo "Usage: $0 [centrifugo|users-service|all]"
        exit 1
        ;;
esac

echo ""
echo "Targets updated. Prometheus will pick up changes within 10 seconds."

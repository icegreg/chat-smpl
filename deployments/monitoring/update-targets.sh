#!/bin/bash
# Update Prometheus targets dynamically based on running containers
# Usage: ./update-targets.sh [service_name]
# Run this after scaling services: docker compose up -d --scale centrifugo=5

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGETS_DIR="$SCRIPT_DIR/targets"

mkdir -p "$TARGETS_DIR"

update_centrifugo_targets() {
    local targets=""
    local first=true

    # Get all centrifugo container names
    for container in $(docker ps --format "{{.Names}}" | grep centrifugo); do
        if [ "$first" = true ]; then
            first=false
        else
            targets="$targets,"
        fi
        targets="$targets\n      \"$container:8000\""
    done

    if [ -n "$targets" ]; then
        cat > "$TARGETS_DIR/centrifugo.json" << EOF
[
  {
    "targets": [$targets
    ],
    "labels": {
      "job": "centrifugo"
    }
  }
]
EOF
        echo "Updated centrifugo targets: $(docker ps --format '{{.Names}}' | grep centrifugo | wc -l) instances"
    else
        echo "No centrifugo containers found"
    fi
}

update_users_service_targets() {
    local targets=""
    local first=true

    for container in $(docker ps --format "{{.Names}}" | grep users-service); do
        if [ "$first" = true ]; then
            first=false
        else
            targets="$targets,"
        fi
        targets="$targets\n      \"$container:8081\""
    done

    if [ -n "$targets" ]; then
        cat > "$TARGETS_DIR/users-service.json" << EOF
[
  {
    "targets": [$targets
    ],
    "labels": {
      "job": "users-service"
    }
  }
]
EOF
        echo "Updated users-service targets: $(docker ps --format '{{.Names}}' | grep users-service | wc -l) instances"
    fi
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

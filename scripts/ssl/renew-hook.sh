#!/bin/bash
#
# Post-renewal hook for Let's Encrypt certificates
# Copies certificates to FreeSWITCH format and reloads services
#
# This script is called by certbot after successful renewal
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

NGINX_CERTS="$PROJECT_ROOT/deployments/nginx/certs"
FREESWITCH_CERTS="$PROJECT_ROOT/deployments/freeswitch/certs"

echo "Post-renewal hook: Updating certificates..."

# Find the renewed certificate (newest in letsencrypt/live)
LETSENCRYPT_LIVE="$NGINX_CERTS/letsencrypt/live"
if [ -d "$LETSENCRYPT_LIVE" ]; then
    DOMAIN_DIR=$(ls -1 "$LETSENCRYPT_LIVE" | head -1)
    if [ -n "$DOMAIN_DIR" ]; then
        CERT_DIR="$LETSENCRYPT_LIVE/$DOMAIN_DIR"

        # Update nginx certs symlinks
        ln -sf "letsencrypt/live/$DOMAIN_DIR/fullchain.pem" "$NGINX_CERTS/fullchain.pem"
        ln -sf "letsencrypt/live/$DOMAIN_DIR/privkey.pem" "$NGINX_CERTS/privkey.pem"

        # Create FreeSWITCH format (combined wss.pem = fullchain + privkey)
        mkdir -p "$FREESWITCH_CERTS"
        cat "$CERT_DIR/fullchain.pem" "$CERT_DIR/privkey.pem" > "$FREESWITCH_CERTS/wss.pem"
        cp "$CERT_DIR/fullchain.pem" "$FREESWITCH_CERTS/wss.crt"
        cp "$CERT_DIR/privkey.pem" "$FREESWITCH_CERTS/wss.key"

        echo "Certificates updated for domain: $DOMAIN_DIR"
    fi
fi

# Reload nginx
echo "Reloading nginx..."
docker exec chatapp-nginx-ssl nginx -s reload 2>/dev/null || true

# Reload FreeSWITCH (if running)
echo "Reloading FreeSWITCH..."
docker exec chatapp-freeswitch fs_cli -x "sofia profile internal restart" 2>/dev/null || true

echo "Certificate renewal complete!"

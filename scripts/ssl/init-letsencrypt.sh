#!/bin/bash
#
# Initial Let's Encrypt certificate setup
#
# Usage:
#   ./init-letsencrypt.sh <domain> <email> [--staging]
#
# Example:
#   ./init-letsencrypt.sh mydomain.com admin@mydomain.com
#   ./init-letsencrypt.sh mydomain.com admin@mydomain.com --staging  # For testing
#

set -e

DOMAIN="$1"
EMAIL="$2"
STAGING="$3"

if [ -z "$DOMAIN" ] || [ -z "$EMAIL" ]; then
    echo "Usage: $0 <domain> <email> [--staging]"
    echo ""
    echo "Example:"
    echo "  $0 mydomain.com admin@mydomain.com"
    echo "  $0 mydomain.com admin@mydomain.com --staging"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "========================================"
echo "Let's Encrypt Certificate Setup"
echo "========================================"
echo "Domain: $DOMAIN"
echo "Email: $EMAIL"
echo "Staging: ${STAGING:-no}"
echo "Project: $PROJECT_ROOT"
echo ""

# Create directories
CERTS_DIR="$PROJECT_ROOT/deployments/nginx/certs/letsencrypt"
WEBROOT_DIR="$PROJECT_ROOT/deployments/nginx/certbot-webroot"

mkdir -p "$CERTS_DIR"
mkdir -p "$WEBROOT_DIR"

# Staging flag for testing
STAGING_ARG=""
if [ "$STAGING" = "--staging" ]; then
    STAGING_ARG="--staging"
    echo "WARNING: Using Let's Encrypt staging environment (certificates won't be trusted)"
fi

# Step 1: Start nginx with temporary self-signed cert for ACME challenge
echo ""
echo "Step 1: Creating temporary self-signed certificate..."
"$SCRIPT_DIR/generate-self-signed.sh" "$DOMAIN"

# Step 2: Start nginx-ssl service (without certbot)
echo ""
echo "Step 2: Starting nginx for ACME challenge..."
cd "$PROJECT_ROOT"

# Start just nginx-ssl with self-signed certs
docker-compose --profile ssl-custom up -d nginx-ssl

echo "Waiting for nginx to start..."
sleep 5

# Step 3: Run certbot to get real certificate
echo ""
echo "Step 3: Obtaining Let's Encrypt certificate..."

docker run --rm \
    -v "$PROJECT_ROOT/deployments/nginx/certs/letsencrypt:/etc/letsencrypt" \
    -v "$WEBROOT_DIR:/var/www/certbot" \
    --network chatapp-network \
    certbot/certbot certonly \
    --webroot \
    --webroot-path=/var/www/certbot \
    --email "$EMAIL" \
    --agree-tos \
    --no-eff-email \
    -d "$DOMAIN" \
    $STAGING_ARG

# Step 4: Copy certificates to nginx certs directory
echo ""
echo "Step 4: Setting up certificate symlinks..."

# Create symlinks from letsencrypt to main certs directory
LIVE_DIR="$CERTS_DIR/live/$DOMAIN"
NGINX_CERTS="$PROJECT_ROOT/deployments/nginx/certs"

if [ -f "$LIVE_DIR/fullchain.pem" ]; then
    ln -sf "letsencrypt/live/$DOMAIN/fullchain.pem" "$NGINX_CERTS/fullchain.pem"
    ln -sf "letsencrypt/live/$DOMAIN/privkey.pem" "$NGINX_CERTS/privkey.pem"
    echo "Certificate symlinks created successfully!"
else
    echo "ERROR: Certificate files not found in $LIVE_DIR"
    exit 1
fi

# Step 5: Create combined cert for FreeSWITCH (wss.pem = fullchain + privkey)
echo ""
echo "Step 5: Creating FreeSWITCH certificate (wss.pem)..."
"$SCRIPT_DIR/renew-hook.sh"

# Step 6: Generate DH parameters (optional, for extra security)
echo ""
echo "Step 6: Generating DH parameters (this may take a while)..."
if [ ! -f "$NGINX_CERTS/dhparam.pem" ]; then
    openssl dhparam -out "$NGINX_CERTS/dhparam.pem" 2048
    echo "DH parameters generated."
else
    echo "DH parameters already exist, skipping."
fi

# Step 7: Restart nginx with real certificate
echo ""
echo "Step 7: Restarting nginx with Let's Encrypt certificate..."
docker-compose --profile ssl-letsencrypt up -d nginx-ssl

echo ""
echo "========================================"
echo "Setup complete!"
echo "========================================"
echo ""
echo "Your site should now be accessible at: https://$DOMAIN"
echo ""
echo "To start all services with SSL:"
echo "  docker-compose --profile ssl-letsencrypt up -d"
echo ""
echo "Certificate will auto-renew via certbot container."

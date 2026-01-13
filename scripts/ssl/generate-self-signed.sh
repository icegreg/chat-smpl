#!/bin/bash
#
# Generate self-signed SSL certificates for development/testing
#
# Usage:
#   ./generate-self-signed.sh [domain]
#
# Example:
#   ./generate-self-signed.sh                    # Uses 'localhost'
#   ./generate-self-signed.sh myapp.local        # Custom domain
#   ./generate-self-signed.sh 192.168.1.100      # IP address
#

set -e

DOMAIN="${1:-localhost}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

CERTS_DIR="$PROJECT_ROOT/deployments/nginx/certs"
FREESWITCH_CERTS="$PROJECT_ROOT/deployments/freeswitch/certs"

echo "========================================"
echo "Self-Signed Certificate Generator"
echo "========================================"
echo "Domain: $DOMAIN"
echo "Output: $CERTS_DIR"
echo ""

mkdir -p "$CERTS_DIR"
mkdir -p "$FREESWITCH_CERTS"

# Generate private key
echo "Generating private key..."
openssl genrsa -out "$CERTS_DIR/privkey.pem" 2048

# Generate certificate signing request with SAN
echo "Generating certificate..."

# Determine if domain is IP address
if [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    SAN="IP:$DOMAIN,IP:127.0.0.1,DNS:localhost"
else
    SAN="DNS:$DOMAIN,DNS:*.$DOMAIN,DNS:localhost,IP:127.0.0.1"
fi

# Create temporary config for SAN
cat > /tmp/openssl-san.cnf << EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
C = RU
ST = Moscow
L = Moscow
O = ChatApp Development
OU = IT
CN = $DOMAIN

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = $SAN
EOF

# Generate self-signed certificate
openssl req -new -x509 \
    -key "$CERTS_DIR/privkey.pem" \
    -out "$CERTS_DIR/fullchain.pem" \
    -days 365 \
    -config /tmp/openssl-san.cnf

rm /tmp/openssl-san.cnf

echo "Certificates generated:"
echo "  - $CERTS_DIR/fullchain.pem"
echo "  - $CERTS_DIR/privkey.pem"

# Create FreeSWITCH format (combined wss.pem)
echo ""
echo "Creating FreeSWITCH certificate..."
cat "$CERTS_DIR/fullchain.pem" "$CERTS_DIR/privkey.pem" > "$FREESWITCH_CERTS/wss.pem"
cp "$CERTS_DIR/fullchain.pem" "$FREESWITCH_CERTS/wss.crt"
cp "$CERTS_DIR/privkey.pem" "$FREESWITCH_CERTS/wss.key"

echo "FreeSWITCH certificates:"
echo "  - $FREESWITCH_CERTS/wss.pem"
echo "  - $FREESWITCH_CERTS/wss.crt"
echo "  - $FREESWITCH_CERTS/wss.key"

# Also copy to old location for backward compatibility
cp "$CERTS_DIR/fullchain.pem" "$CERTS_DIR/wss.crt" 2>/dev/null || true
cp "$CERTS_DIR/privkey.pem" "$CERTS_DIR/wss.key" 2>/dev/null || true

echo ""
echo "========================================"
echo "Self-signed certificate generated!"
echo "========================================"
echo ""
echo "To use with custom certificates profile:"
echo "  docker-compose --profile ssl-custom up -d"
echo ""
echo "NOTE: Self-signed certificates will show browser warnings."
echo "      For production, use Let's Encrypt or a trusted CA."

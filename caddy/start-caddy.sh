#!/bin/bash
set -e

if [ "${DATABASUS_TLS_ENABLED:-false}" != "true" ]; then
    exit 0
fi

echo "Starting Caddy HTTPS proxy on port 4006..."
mkdir -p /databasus-data/caddy/data /databasus-data/caddy/config
chown -R databasus:databasus /databasus-data/caddy

if [ -n "${DATABASUS_TLS_CERT_PATH:-}" ] || [ -n "${DATABASUS_TLS_KEY_PATH:-}" ]; then
    if [ -z "${DATABASUS_TLS_CERT_PATH:-}" ] || [ -z "${DATABASUS_TLS_KEY_PATH:-}" ]; then
        echo "ERROR: DATABASUS_TLS_CERT_PATH and DATABASUS_TLS_KEY_PATH must both be set." >&2
        exit 1
    fi

    if ! gosu databasus test -r "$DATABASUS_TLS_CERT_PATH"; then
        echo "ERROR: TLS certificate is not readable: $DATABASUS_TLS_CERT_PATH" >&2
        exit 1
    fi

    if ! gosu databasus test -r "$DATABASUS_TLS_KEY_PATH"; then
        echo "ERROR: TLS private key is not readable: $DATABASUS_TLS_KEY_PATH" >&2
        exit 1
    fi

    echo "Using custom TLS certificate."
    CADDYFILE=/app/Caddyfile.custom-cert
else
    if [ ! -s /databasus-data/caddy/self-signed.crt ] || [ ! -s /databasus-data/caddy/self-signed.key ]; then
        echo "Generating self-signed TLS certificate..."
        gosu databasus openssl req \
          -x509 \
          -newkey rsa:2048 \
          -nodes \
          -sha256 \
          -days 3650 \
          -subj "/CN=localhost" \
          -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
          -keyout /databasus-data/caddy/self-signed.key \
          -out /databasus-data/caddy/self-signed.crt
        chmod 600 /databasus-data/caddy/self-signed.key
    fi

    echo "Using generated self-signed TLS certificate."
    CADDYFILE=/app/Caddyfile
fi

run_caddy() {
    gosu databasus env \
      XDG_DATA_HOME=/databasus-data/caddy/data \
      XDG_CONFIG_HOME=/databasus-data/caddy/config \
      caddy "$@"
}

run_caddy validate --config "$CADDYFILE" --adapter caddyfile
run_caddy run --config "$CADDYFILE" --adapter caddyfile &

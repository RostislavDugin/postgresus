#!/bin/bash
set -e

# Check for legacy postgresus-data volume mount
if [ -d "/postgresus-data" ] && [ "$(ls -A /postgresus-data 2>/dev/null)" ]; then
    echo ""
    echo "=========================================="
    echo "ERROR: Legacy volume detected!"
    echo "=========================================="
    echo ""
    echo "You are using the \`postgresus-data\` folder. It seems you changed the image name from Postgresus to Databasus without changing the volume."
    echo ""
    echo "Please either:"
    echo "  1. Switch back to image rostislavdugin/postgresus:latest (supported until ~Dec 2026)"
    echo "  2. Read the migration guide: https://databasus.com/installation/#postgresus-migration"
    echo ""
    echo "=========================================="
    exit 1
fi

# ========= Adjust postgres user UID/GID =========
PUID=${PUID:-999}
PGID=${PGID:-999}

CURRENT_UID=$(id -u postgres)
CURRENT_GID=$(id -g postgres)

if [ "$CURRENT_GID" != "$PGID" ]; then
    echo "Adjusting postgres group GID from $CURRENT_GID to $PGID..."
    groupmod -o -g "$PGID" postgres
fi

if [ "$CURRENT_UID" != "$PUID" ]; then
    echo "Adjusting postgres user UID from $CURRENT_UID to $PUID..."
    usermod -o -u "$PUID" postgres
fi

# PostgreSQL 17 binary paths
PG_BIN="/usr/lib/postgresql/17/bin"

# Generate runtime configuration for frontend
echo "Generating runtime configuration..."

# Detect if email is configured (both SMTP_HOST and DATABASUS_URL must be set)
if [ -n "${SMTP_HOST:-}" ] && [ -n "${DATABASUS_URL:-}" ]; then
  IS_EMAIL_CONFIGURED="true"
else
  IS_EMAIL_CONFIGURED="false"
fi

GENERIC_PROVIDERS_JSON="["
first=true
for var in $(env | grep -E '^GENERIC_OAUTH_.*_CLIENT_ID=' | sed 's/=.*//'); do
  key=$(echo "$var" | sed 's/^GENERIC_OAUTH_//;s/_CLIENT_ID$//')
  name=$(echo "$key" | tr '[:upper:]' '[:lower:]')
  client_id="${!var}"
  auth_url_var="GENERIC_OAUTH_${key}_AUTH_URL"
  auth_url="${!auth_url_var}"
  well_known_var="GENERIC_OAUTH_${key}_WELL_KNOWN"
  well_known="${!well_known_var}"
  scopes_var="GENERIC_OAUTH_${key}_SCOPES"
  scopes="${!scopes_var}"
  [ -z "$scopes" ] && scopes="openid email profile"
  display_var="GENERIC_OAUTH_${key}_DISPLAY_NAME"
  display_name="${!display_var}"
  [ -z "$display_name" ] && display_name="$name"

  if [ -z "$auth_url" ] && [ -n "$well_known" ]; then
    auth_url=$(curl -sf "$well_known" | grep -o '"authorization_endpoint":"[^"]*"' | cut -d'"' -f4)
  fi

  [ -z "$auth_url" ] && continue

  [ "$first" = "false" ] && GENERIC_PROVIDERS_JSON="$GENERIC_PROVIDERS_JSON,"
  GENERIC_PROVIDERS_JSON="${GENERIC_PROVIDERS_JSON}{\"name\":\"$name\",\"displayName\":\"$display_name\",\"clientId\":\"$client_id\",\"authUrl\":\"$auth_url\",\"scopes\":\"$scopes\"}"
  first=false
done
GENERIC_PROVIDERS_JSON="${GENERIC_PROVIDERS_JSON}]"

cat > /app/ui/build/runtime-config.js <<JSEOF
// Runtime configuration injected at container startup
// This file is generated dynamically and should not be edited manually
window.__RUNTIME_CONFIG__ = {
  GITHUB_CLIENT_ID: '${GITHUB_CLIENT_ID:-}',
  GOOGLE_CLIENT_ID: '${GOOGLE_CLIENT_ID:-}',
  IS_EMAIL_CONFIGURED: '$IS_EMAIL_CONFIGURED',
  CLOUDFLARE_TURNSTILE_SITE_KEY: '${CLOUDFLARE_TURNSTILE_SITE_KEY:-}',
  CONTAINER_ARCH: '${CONTAINER_ARCH:-unknown}',
  GENERIC_OAUTH_PROVIDERS: '$GENERIC_PROVIDERS_JSON'
};
JSEOF

# Inject analytics script if provided (only if not already injected)
if [ -n "${ANALYTICS_SCRIPT:-}" ]; then
  if ! grep -q "rybbit.databasus.com" /app/ui/build/index.html 2>/dev/null; then
    echo "Injecting analytics script..."
    sed -i "s#</head>#  ${ANALYTICS_SCRIPT}\
  </head>#" /app/ui/build/index.html
  fi
fi

# Ensure proper ownership of data directory
echo "Setting up data directory permissions..."
mkdir -p /databasus-data/pgdata
mkdir -p /databasus-data/temp
mkdir -p /databasus-data/backups
chown databasus:databasus /databasus-data
chown -R postgres:postgres /databasus-data/pgdata
chown -R databasus:databasus /databasus-data/temp /databasus-data/backups
# Upgrade path: secret.key and instance.json may be owned by root or postgres
# from older images. Re-own them so the non-root main process can read/write.
chown databasus:databasus /databasus-data/secret.key /databasus-data/instance.json 2>/dev/null || true
chmod 700 /databasus-data/temp

# ========= Start Valkey (internal cache) =========
echo "Configuring Valkey cache..."
cat > /tmp/valkey.conf << 'VALKEY_CONFIG'
port 6379
bind 127.0.0.1
protected-mode yes
save ""
maxmemory 256mb
maxmemory-policy allkeys-lru
VALKEY_CONFIG

echo "Starting Valkey..."
valkey-server /tmp/valkey.conf &
VALKEY_PID=$!

echo "Waiting for Valkey to be ready..."
for i in {1..30}; do
    if valkey-cli ping >/dev/null 2>&1; then
        echo "Valkey is ready!"
        break
    fi
    sleep 1
done

# Initialize PostgreSQL if not already initialized
if [ ! -s "/databasus-data/pgdata/PG_VERSION" ]; then
    echo "Initializing PostgreSQL database..."
    gosu postgres $PG_BIN/initdb -D /databasus-data/pgdata --encoding=UTF8 --locale=C.UTF-8

    # Configure PostgreSQL
    echo "host all all 127.0.0.1/32 md5" >> /databasus-data/pgdata/pg_hba.conf
    echo "local all all trust" >> /databasus-data/pgdata/pg_hba.conf
    echo "port = 5437" >> /databasus-data/pgdata/postgresql.conf
    echo "listen_addresses = 'localhost'" >> /databasus-data/pgdata/postgresql.conf
    echo "shared_buffers = 256MB" >> /databasus-data/pgdata/postgresql.conf
    echo "max_connections = 100" >> /databasus-data/pgdata/postgresql.conf
fi

# Function to start PostgreSQL and wait for it to be ready
start_postgres() {
    echo "Starting PostgreSQL..."
    # -k /tmp: create Unix socket and lock file in /tmp instead of /var/run/postgresql/.
    # On NAS systems (e.g. TrueNAS Scale), the ZFS-backed Docker overlay filesystem
    # ignores chown/chmod on directories from image layers, so PostgreSQL gets
    # "Permission denied" when creating .s.PGSQL.5437.lock in /var/run/postgresql/.
    # All internal connections use TCP (-h localhost), so the socket location does not matter.
    gosu postgres $PG_BIN/postgres -D /databasus-data/pgdata -p 5437 -k /tmp &
    POSTGRES_PID=$!

    echo "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if gosu postgres $PG_BIN/pg_isready -p 5437 -h localhost >/dev/null 2>&1; then
            echo "PostgreSQL is ready!"
            return 0
        fi
        sleep 1
    done
    return 1
}

# Try to start PostgreSQL
if ! start_postgres; then
    echo ""
    echo "=========================================="
    echo "PostgreSQL failed to start. Attempting WAL reset recovery..."
    echo "=========================================="
    echo ""

    # Kill any remaining postgres processes
    pkill -9 postgres 2>/dev/null || true
    sleep 2

    # Attempt pg_resetwal to recover from WAL corruption
    echo "Running pg_resetwal to reset WAL..."
    if gosu postgres $PG_BIN/pg_resetwal -f /databasus-data/pgdata; then
        echo "WAL reset successful. Restarting PostgreSQL..."

        # Try starting PostgreSQL again after WAL reset
        if start_postgres; then
            echo "PostgreSQL recovered successfully after WAL reset!"
        else
            echo ""
            echo "=========================================="
            echo "ERROR: PostgreSQL failed to start even after WAL reset."
            echo "The database may be severely corrupted."
            echo ""
            echo "Options:"
            echo "  1. Delete the volume and start fresh (data loss)"
            echo "  2. Manually inspect /databasus-data/pgdata for issues"
            echo "=========================================="
            exit 1
        fi
    else
        echo ""
        echo "=========================================="
        echo "ERROR: pg_resetwal failed."
        echo "The database may be severely corrupted."
        echo ""
        echo "Options:"
        echo "  1. Delete the volume and start fresh (data loss)"
        echo "  2. Manually inspect /databasus-data/pgdata for issues"
        echo "=========================================="
        exit 1
    fi
fi

# Create database and set password for postgres user
echo "Setting up database and user..."
gosu postgres $PG_BIN/psql -p 5437 -h localhost -d postgres << 'SQL'

-- We use stub password, because internal DB is not exposed outside container
ALTER USER postgres WITH PASSWORD 'Q1234567';
SELECT 'CREATE DATABASE databasus OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'databasus')
\gexec
\q
SQL

# ========= Refuse to start if legacy WAL backup data exists =========
# The agent-based WAL_V1 backup type was removed. Existing installs with
# WAL-mode databases must downgrade, manually remove them and then upgrade.
echo "Checking for legacy WAL backup configuration..."
WAL_CHECK_DSN="postgres://postgres:Q1234567@localhost:5437/databasus"

WAL_CHECK_COL=$(gosu databasus $PG_BIN/psql "$WAL_CHECK_DSN" -tA -c "SELECT 1 FROM information_schema.columns WHERE table_name='postgresql_databases' AND column_name='backup_type' LIMIT 1" 2>/dev/null || true)

if [ "$WAL_CHECK_COL" = "1" ]; then
    WAL_CHECK_ROW=$(gosu databasus $PG_BIN/psql "$WAL_CHECK_DSN" -tA -c "SELECT 1 FROM postgresql_databases WHERE backup_type='WAL_V1' LIMIT 1" 2>/dev/null || true)
    if [ "$WAL_CHECK_ROW" = "1" ]; then
        echo ""
        echo "=========================================="
        echo "ERROR: Agent (WAL_V1) backup approach is no longer supported."
        echo "=========================================="
        echo ""
        echo "Please downgrade to version 3.42.0, remove all WAL-mode databases"
        echo "manually and then upgrade again. This safeguard exists to avoid"
        echo "corrupting already-set-up agents."
        echo ""
        echo "=========================================="
        exit 1
    fi
fi
echo "No legacy WAL backup data detected."

# Start the main application
echo "Starting Databasus application..."

exec gosu databasus ./main

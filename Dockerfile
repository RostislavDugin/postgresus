# ========= BUILD FRONTEND =========
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-build

WORKDIR /frontend

# Add version for the frontend build
ARG APP_VERSION=dev
ENV VITE_APP_VERSION=$APP_VERSION

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./

# Copy .env file (with fallback to .env.production.example)
RUN if [ ! -f .env ]; then \
  if [ -f .env.production.example ]; then \
  cp .env.production.example .env; \
  fi; \
  fi

RUN npm run build

# ========= BUILD BACKEND =========
# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.26.1 AS backend-build

# Make TARGET args available early so tools built here match the final image arch
ARG TARGETOS
ARG TARGETARCH

# Install Go public tools needed in runtime. Use `go build` for goose so the
# binary is compiled for the target architecture instead of downloading a
# prebuilt binary which may have the wrong architecture (causes exec format
# errors on ARM).
RUN git clone --depth 1 --branch v3.24.3 https://github.com/pressly/goose.git /tmp/goose && \
  cd /tmp/goose/cmd/goose && \
  GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
  go build -o /usr/local/bin/goose . && \
  rm -rf /tmp/goose
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Set working directory
WORKDIR /app

# Install Go dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Create required directories for embedding
RUN mkdir -p /app/ui/build

# Copy frontend build output for embedding
COPY --from=frontend-build /frontend/dist /app/ui/build

# Generate Swagger documentation
COPY backend/ ./
RUN swag init -d . -g cmd/main.go -o swagger

# Compile the backend
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN CGO_ENABLED=0 \
  GOOS=$TARGETOS \
  GOARCH=$TARGETARCH \
  go build -o /app/main ./cmd/main.go


# ========= BUILD AGENT =========
# Builds the databasus-agent CLI binary for BOTH x86_64 and ARM64.
# Both architectures are always built because:
# - Databasus server runs on one arch (e.g. amd64)
# - The agent runs on remote PostgreSQL servers that may be on a
#   different arch (e.g. arm64)
# - The backend serves the correct binary based on the agent's
#   ?arch= query parameter
#
# We cross-compile from the build platform (no QEMU needed) because the
# agent is pure Go with zero C dependencies.
# CGO_ENABLED=0 produces fully static binaries — no glibc/musl dependency,
# so the agent runs on any Linux distro (Alpine, Debian, Ubuntu, RHEL, etc.).
# APP_VERSION is baked into the binary via -ldflags so the agent can
# compare its version against the server and auto-update when needed.
FROM --platform=$BUILDPLATFORM golang:1.26.1 AS agent-build

ARG APP_VERSION=dev

WORKDIR /agent

COPY agent/go.mod ./
RUN go mod download

COPY agent/ ./

# Build for x86_64 (amd64) — static binary, no glibc dependency
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /agent-binaries/databasus-agent-linux-amd64 ./cmd/main.go

# Build for ARM64 (arm64) — static binary, no glibc dependency
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /agent-binaries/databasus-agent-linux-arm64 ./cmd/main.go


# ========= RUNTIME =========
FROM debian:bookworm-slim

# Add version metadata to runtime image
ARG APP_VERSION=dev
ARG TARGETARCH
LABEL org.opencontainers.image.version=$APP_VERSION
ENV APP_VERSION=$APP_VERSION
ENV CONTAINER_ARCH=$TARGETARCH

# Set production mode for Docker containers
ENV ENV_MODE=production

# ========= STEP 1: Install base packages =========
RUN apt-get update
RUN apt-get install -y --no-install-recommends \
  wget ca-certificates gnupg lsb-release sudo gosu curl unzip xz-utils libncurses5 libncurses6
RUN rm -rf /var/lib/apt/lists/*

# ========= Install PostgreSQL client binaries (versions 12-18) =========
# Pre-downloaded binaries from assets/tools/ - no network download needed
ARG TARGETARCH
RUN mkdir -p /usr/lib/postgresql/12/bin /usr/lib/postgresql/13/bin \
  /usr/lib/postgresql/14/bin /usr/lib/postgresql/15/bin \
  /usr/lib/postgresql/16/bin /usr/lib/postgresql/17/bin \
  /usr/lib/postgresql/18/bin

# Copy pre-downloaded PostgreSQL binaries based on architecture
COPY assets/tools/x64/postgresql/ /tmp/pg-x64/
COPY assets/tools/arm/postgresql/ /tmp/pg-arm/
RUN if [ "$TARGETARCH" = "amd64" ]; then \
  cp -r /tmp/pg-x64/postgresql-12/bin/* /usr/lib/postgresql/12/bin/ && \
  cp -r /tmp/pg-x64/postgresql-13/bin/* /usr/lib/postgresql/13/bin/ && \
  cp -r /tmp/pg-x64/postgresql-14/bin/* /usr/lib/postgresql/14/bin/ && \
  cp -r /tmp/pg-x64/postgresql-15/bin/* /usr/lib/postgresql/15/bin/ && \
  cp -r /tmp/pg-x64/postgresql-16/bin/* /usr/lib/postgresql/16/bin/ && \
  cp -r /tmp/pg-x64/postgresql-17/bin/* /usr/lib/postgresql/17/bin/ && \
  cp -r /tmp/pg-x64/postgresql-18/bin/* /usr/lib/postgresql/18/bin/; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  cp -r /tmp/pg-arm/postgresql-12/bin/* /usr/lib/postgresql/12/bin/ && \
  cp -r /tmp/pg-arm/postgresql-13/bin/* /usr/lib/postgresql/13/bin/ && \
  cp -r /tmp/pg-arm/postgresql-14/bin/* /usr/lib/postgresql/14/bin/ && \
  cp -r /tmp/pg-arm/postgresql-15/bin/* /usr/lib/postgresql/15/bin/ && \
  cp -r /tmp/pg-arm/postgresql-16/bin/* /usr/lib/postgresql/16/bin/ && \
  cp -r /tmp/pg-arm/postgresql-17/bin/* /usr/lib/postgresql/17/bin/ && \
  cp -r /tmp/pg-arm/postgresql-18/bin/* /usr/lib/postgresql/18/bin/; \
  fi && \
  rm -rf /tmp/pg-x64 /tmp/pg-arm && \
  chmod +x /usr/lib/postgresql/*/bin/*

# Install PostgreSQL 17 server (needed for internal database)
# Add PostgreSQL repository for server installation only
RUN wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
  echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
  > /etc/apt/sources.list.d/pgdg.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends postgresql-17 && \
  rm -rf /var/lib/apt/lists/*

# Install Valkey server from debian repository
# Valkey is only accessible internally (localhost) - not exposed outside container
RUN wget -O /usr/share/keyrings/greensec.github.io-valkey-debian.key https://greensec.github.io/valkey-debian/public.key && \
  echo "deb [signed-by=/usr/share/keyrings/greensec.github.io-valkey-debian.key] https://greensec.github.io/valkey-debian/repo $(lsb_release -cs) main" \
  > /etc/apt/sources.list.d/valkey-debian.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends valkey && \
  rm -rf /var/lib/apt/lists/*

# ========= Install rclone =========
RUN apt-get update && \
  apt-get install -y --no-install-recommends rclone && \
  rm -rf /var/lib/apt/lists/*

# Create directories for all database clients
RUN mkdir -p /usr/local/mysql-5.7/bin /usr/local/mysql-8.0/bin /usr/local/mysql-8.4/bin \
  /usr/local/mysql-9/bin \
  /usr/local/mariadb-10.6/bin /usr/local/mariadb-12.1/bin \
  /usr/local/mongodb-database-tools/bin

# ========= Install MySQL clients (5.7, 8.0, 8.4, 9) =========
# Pre-downloaded binaries from assets/tools/ - no network download needed
# Note: MySQL 5.7 is only available for x86_64
# Note: MySQL binaries require libncurses5 for terminal handling
COPY assets/tools/x64/mysql/ /tmp/mysql-x64/
COPY assets/tools/arm/mysql/ /tmp/mysql-arm/
RUN if [ "$TARGETARCH" = "amd64" ]; then \
  cp /tmp/mysql-x64/mysql-5.7/bin/* /usr/local/mysql-5.7/bin/ && \
  cp /tmp/mysql-x64/mysql-8.0/bin/* /usr/local/mysql-8.0/bin/ && \
  cp /tmp/mysql-x64/mysql-8.4/bin/* /usr/local/mysql-8.4/bin/ && \
  cp /tmp/mysql-x64/mysql-9/bin/* /usr/local/mysql-9/bin/; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  echo "MySQL 5.7 not available for arm64, skipping..." && \
  cp /tmp/mysql-arm/mysql-8.0/bin/* /usr/local/mysql-8.0/bin/ && \
  cp /tmp/mysql-arm/mysql-8.4/bin/* /usr/local/mysql-8.4/bin/ && \
  cp /tmp/mysql-arm/mysql-9/bin/* /usr/local/mysql-9/bin/; \
  fi && \
  rm -rf /tmp/mysql-x64 /tmp/mysql-arm && \
  chmod +x /usr/local/mysql-*/bin/*

# ========= Install MariaDB clients (10.6, 12.1) =========
# Pre-downloaded binaries from assets/tools/ - no network download needed
# 10.6 (legacy): For older servers (5.5, 10.1) that don't have generation_expression column
# 12.1 (modern): For newer servers (10.2+)
COPY assets/tools/x64/mariadb/ /tmp/mariadb-x64/
COPY assets/tools/arm/mariadb/ /tmp/mariadb-arm/
RUN if [ "$TARGETARCH" = "amd64" ]; then \
  cp /tmp/mariadb-x64/mariadb-10.6/bin/* /usr/local/mariadb-10.6/bin/ && \
  cp /tmp/mariadb-x64/mariadb-12.1/bin/* /usr/local/mariadb-12.1/bin/; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  cp /tmp/mariadb-arm/mariadb-10.6/bin/* /usr/local/mariadb-10.6/bin/ && \
  cp /tmp/mariadb-arm/mariadb-12.1/bin/* /usr/local/mariadb-12.1/bin/; \
  fi && \
  rm -rf /tmp/mariadb-x64 /tmp/mariadb-arm && \
  chmod +x /usr/local/mariadb-*/bin/*

# ========= Install MongoDB Database Tools =========
# Note: MongoDB Database Tools are backward compatible - single version supports all server versions (4.0-8.0)
# Note: For ARM64, we use Ubuntu 22.04 package as MongoDB doesn't provide Debian 12 ARM64 packages
RUN apt-get update && \
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-x86_64-100.10.0.deb -O /tmp/mongodb-database-tools.deb; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  wget -q https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2204-arm64-100.10.0.deb -O /tmp/mongodb-database-tools.deb; \
  fi && \
  dpkg -i /tmp/mongodb-database-tools.deb || apt-get install -f -y --no-install-recommends && \
  rm -f /tmp/mongodb-database-tools.deb && \
  rm -rf /var/lib/apt/lists/* && \
  mkdir -p /usr/local/mongodb-database-tools/bin && \
  if [ -f /usr/bin/mongodump ]; then \
  ln -sf /usr/bin/mongodump /usr/local/mongodb-database-tools/bin/mongodump; \
  fi && \
  if [ -f /usr/bin/mongorestore ]; then \
  ln -sf /usr/bin/mongorestore /usr/local/mongodb-database-tools/bin/mongorestore; \
  fi

# ========= Install ClickHouse Client =========
# clickhouse-client is the ClickHouse multicall binary invoked via symlink; it speaks
# the native protocol and is forward + backward compatible with ClickHouse server
# versions 23.x through 26.x. Pinned to v25.8 LTS. The release-published .sha512
# sidecar is used by sha512sum -c to verify the tarball before extraction.
ARG CLICKHOUSE_VERSION=25.8.22.28
RUN set -eux; \
  arch_suffix=$(case "$TARGETARCH" in amd64) echo amd64 ;; arm64) echo arm64 ;; *) echo "unsupported arch $TARGETARCH" >&2; exit 1 ;; esac); \
  tarball="clickhouse-common-static-${CLICKHOUSE_VERSION}-${arch_suffix}.tgz"; \
  base="https://github.com/ClickHouse/ClickHouse/releases/download/v${CLICKHOUSE_VERSION}-lts/${tarball}"; \
  cd /tmp; \
  wget -q "$base" -O "$tarball"; \
  wget -q "${base}.sha512" -O "${tarball}.sha512"; \
  sha512sum -c "${tarball}.sha512"; \
  tar -xzf "$tarball" -C /tmp; \
  install -m 0755 "/tmp/clickhouse-common-static-${CLICKHOUSE_VERSION}/usr/bin/clickhouse" /usr/bin/clickhouse; \
  mkdir -p /usr/local/clickhouse/bin; \
  ln -sf /usr/bin/clickhouse /usr/local/clickhouse/bin/clickhouse-client; \
  rm -rf "/tmp/clickhouse-common-static-${CLICKHOUSE_VERSION}" "/tmp/${tarball}" "/tmp/${tarball}.sha512"

# Create postgres user and set up directories
RUN groupadd -g 999 postgres || true && \
  useradd -m -s /bin/bash -u 999 -g 999 postgres || true && \
  mkdir -p /databasus-data/pgdata && \
  chown -R postgres:postgres /databasus-data/pgdata

WORKDIR /app

# Copy Goose from build stage
COPY --from=backend-build /usr/local/bin/goose /usr/local/bin/goose

# Copy app binary 
COPY --from=backend-build /app/main .

# Copy migrations directory
COPY backend/migrations ./migrations

# Copy UI files
COPY --from=backend-build /app/ui/build ./ui/build

# Copy cloud static HTML template (injected into index.html at startup when IS_CLOUD=true)
COPY frontend/cloud-root-content.html /app/cloud-root-content.html

# Copy agent binaries (both architectures) — served by the backend
# at GET /api/v1/system/agent?arch=amd64|arm64
COPY --from=agent-build /agent-binaries ./agent-binaries

# Copy .env file (with fallback to .env.production.example)
COPY backend/.env* /app/
RUN if [ ! -f /app/.env ]; then \
  if [ -f /app/.env.production.example ]; then \
  cp /app/.env.production.example /app/.env; \
  fi; \
  fi

# Create startup script
COPY <<EOF /app/start.sh
#!/bin/bash
set -e

# Check for legacy postgresus-data volume mount
if [ -d "/postgresus-data" ] && [ "\$(ls -A /postgresus-data 2>/dev/null)" ]; then
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
PUID=\${PUID:-999}
PGID=\${PGID:-999}

CURRENT_UID=\$(id -u postgres)
CURRENT_GID=\$(id -g postgres)

if [ "\$CURRENT_GID" != "\$PGID" ]; then
    echo "Adjusting postgres group GID from \$CURRENT_GID to \$PGID..."
    groupmod -o -g "\$PGID" postgres
fi

if [ "\$CURRENT_UID" != "\$PUID" ]; then
    echo "Adjusting postgres user UID from \$CURRENT_UID to \$PUID..."
    usermod -o -u "\$PUID" postgres
fi

# PostgreSQL 17 binary paths
PG_BIN="/usr/lib/postgresql/17/bin"

# Generate runtime configuration for frontend
echo "Generating runtime configuration..."

# Detect if email is configured (both SMTP_HOST and DATABASUS_URL must be set)
if [ -n "\${SMTP_HOST:-}" ] && [ -n "\${DATABASUS_URL:-}" ]; then
  IS_EMAIL_CONFIGURED="true"
else
  IS_EMAIL_CONFIGURED="false"
fi

cat > /app/ui/build/runtime-config.js <<JSEOF
// Runtime configuration injected at container startup
// This file is generated dynamically and should not be edited manually
window.__RUNTIME_CONFIG__ = {
  IS_CLOUD: '\${IS_CLOUD:-false}',
  GITHUB_CLIENT_ID: '\${GITHUB_CLIENT_ID:-}',
  GOOGLE_CLIENT_ID: '\${GOOGLE_CLIENT_ID:-}',
  IS_EMAIL_CONFIGURED: '\$IS_EMAIL_CONFIGURED',
  CLOUDFLARE_TURNSTILE_SITE_KEY: '\${CLOUDFLARE_TURNSTILE_SITE_KEY:-}',
  CONTAINER_ARCH: '\${CONTAINER_ARCH:-unknown}',
  CLOUD_PRICE_PER_GB: '\${CLOUD_PRICE_PER_GB:-}',
  CLOUD_PADDLE_CLIENT_TOKEN: '\${CLOUD_PADDLE_CLIENT_TOKEN:-}'
};
JSEOF

# Inject analytics script if provided (only if not already injected)
if [ -n "\${ANALYTICS_SCRIPT:-}" ]; then
  if ! grep -q "rybbit.databasus.com" /app/ui/build/index.html 2>/dev/null; then
    echo "Injecting analytics script..."
    sed -i "s#</head>#  \${ANALYTICS_SCRIPT}\\
  </head>#" /app/ui/build/index.html
  fi
fi

# Inject Paddle script if client token is provided (only if not already injected)
if [ -n "\${CLOUD_PADDLE_CLIENT_TOKEN:-}" ]; then
  if ! grep -q "cdn.paddle.com" /app/ui/build/index.html 2>/dev/null; then
    echo "Injecting Paddle script..."
    sed -i "s#</head>#  <script src=\"https://cdn.paddle.com/paddle/v2/paddle.js\"></script>\\
  </head>#" /app/ui/build/index.html
  fi
fi

# Inject static HTML into root div for cloud mode (payment system requires visible legal links)
if [ "\${IS_CLOUD:-false}" = "true" ]; then
  if ! grep -q "cloud-static-content" /app/ui/build/index.html 2>/dev/null; then
    echo "Injecting cloud static HTML content..."
    perl -i -pe '
      BEGIN {
        open my \$fh, "<", "/app/cloud-root-content.html" or die;
        local \$/;
        \$c = <\$fh>;
        close \$fh;
        \$c =~ s/\\n/ /g;
      }
      s/<div id="root"><\\/div>/<div id="root"><!-- cloud-static-content --><noscript>\$c<\\/noscript><\\/div>/
    ' /app/ui/build/index.html
  fi
fi

# Ensure proper ownership of data directory
echo "Setting up data directory permissions..."
mkdir -p /databasus-data/pgdata
mkdir -p /databasus-data/temp
mkdir -p /databasus-data/backups
chown -R postgres:postgres /databasus-data
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
VALKEY_PID=\$!

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
    gosu postgres \$PG_BIN/initdb -D /databasus-data/pgdata --encoding=UTF8 --locale=C.UTF-8
    
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
    gosu postgres \$PG_BIN/postgres -D /databasus-data/pgdata -p 5437 -k /tmp &
    POSTGRES_PID=\$!
    
    echo "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if gosu postgres \$PG_BIN/pg_isready -p 5437 -h localhost >/dev/null 2>&1; then
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
    if gosu postgres \$PG_BIN/pg_resetwal -f /databasus-data/pgdata; then
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
gosu postgres \$PG_BIN/psql -p 5437 -h localhost -d postgres << 'SQL'

-- We use stub password, because internal DB is not exposed outside container
ALTER USER postgres WITH PASSWORD 'Q1234567';
SELECT 'CREATE DATABASE databasus OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'databasus')
\\gexec
\\q
SQL

# Start the main application
echo "Starting Databasus application..."

# Check and warn about external database/Valkey usage
if [ -n "\${DANGEROUS_EXTERNAL_DATABASE_DSN:-}" ]; then
    echo ""
    echo "=========================================="
    echo "WARNING: Using external database"
    echo "=========================================="
    echo "DANGEROUS_EXTERNAL_DATABASE_DSN is set."
    echo "Application will connect to external PostgreSQL instead of internal instance."
    echo "Internal PostgreSQL is still running in the background."
    echo "=========================================="
    echo ""
fi

if [ -n "\${DANGEROUS_VALKEY_HOST:-}" ]; then
    echo ""
    echo "=========================================="
    echo "WARNING: Using external Valkey"
    echo "=========================================="
    echo "DANGEROUS_VALKEY_HOST is set."
    echo "Application will connect to external Valkey instead of internal instance."
    echo "Internal Valkey is still running in the background."
    echo "=========================================="
    echo ""
fi

exec ./main
EOF

LABEL org.opencontainers.image.source="https://github.com/databasus/databasus"

RUN chmod +x /app/start.sh

EXPOSE 4005

# Volume for PostgreSQL data
VOLUME ["/databasus-data"]

ENTRYPOINT ["/app/start.sh"]
CMD []

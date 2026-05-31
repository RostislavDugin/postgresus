# ========= BUILD FRONTEND =========
FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-build

WORKDIR /frontend

# Add version for the frontend build
ARG APP_VERSION=dev
ENV VITE_APP_VERSION=$APP_VERSION

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ ./

# Copy .env file (with fallback to .env.production.example)
RUN if [ ! -f .env ]; then \
  if [ -f .env.production.example ]; then \
  cp .env.production.example .env; \
  fi; \
  fi

RUN pnpm build

# ========= BUILD BACKEND =========
# Backend build stage
FROM --platform=$BUILDPLATFORM golang:1.26.3 AS backend-build

# Make TARGET args available early so tools built here match the final image arch
ARG TARGETOS
ARG TARGETARCH

# Install Go public tools needed in runtime. Use `go build` for goose so the
# binary is compiled for the target architecture instead of downloading a
# prebuilt binary which may have the wrong architecture (causes exec format
# errors on ARM).
RUN git clone --depth 1 --branch v3.27.1 https://github.com/pressly/goose.git /tmp/goose && \
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
FROM --platform=$BUILDPLATFORM golang:1.26.3 AS agent-build

ARG APP_VERSION=dev

WORKDIR /agent

COPY agent/backup/go.mod ./
RUN go mod download

COPY agent/backup/ ./

# Build for x86_64 (amd64) — static binary, no glibc dependency
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /agent-binaries/databasus-agent-linux-amd64 ./cmd/main.go

# Build for ARM64 (arm64) — static binary, no glibc dependency
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /agent-binaries/databasus-agent-linux-arm64 ./cmd/main.go


# ========= BUILD VERIFICATION AGENT =========
FROM --platform=$BUILDPLATFORM golang:1.26.3 AS verification-agent-build

ARG APP_VERSION=dev

WORKDIR /agent

COPY agent/verification/go.mod agent/verification/go.sum ./
RUN go mod download

COPY agent/verification/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /verification-agent-binaries/databasus-verification-agent-linux-amd64 ./cmd

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -ldflags "-X main.Version=${APP_VERSION}" \
    -o /verification-agent-binaries/databasus-verification-agent-linux-arm64 ./cmd


# ========= CADDY =========
FROM caddy:2-alpine AS caddy


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

# ========= Install all apt packages in a single layer =========
# Combines base packages + PostgreSQL 17 (pgdg repo) + Valkey (greensec repo) + rclone
# into one RUN to minimise layer count and cache-export overhead.
# Valkey is only accessible internally (localhost) — not exposed outside container.
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      wget ca-certificates gnupg lsb-release sudo gosu curl unzip xz-utils openssl \
      libncurses5 libncurses6 rclone \
      libmariadb3 \
      libgnutls30; \
    wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -; \
    echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
      > /etc/apt/sources.list.d/pgdg.list; \
    wget -O /usr/share/keyrings/greensec.github.io-valkey-debian.key \
      https://greensec.github.io/valkey-debian/public.key; \
    echo "deb [signed-by=/usr/share/keyrings/greensec.github.io-valkey-debian.key] https://greensec.github.io/valkey-debian/repo $(lsb_release -cs) main" \
      > /etc/apt/sources.list.d/valkey-debian.list; \
    apt-get update; \
    apt-get install -y --no-install-recommends postgresql-17 valkey; \
    rm -rf /var/lib/apt/lists/*

# ========= Pre-built DB client binaries (PG, MySQL, MariaDB, MongoDB) =========
# All client tools live under /app/assets/tools/<arch>/ — the backend resolves
# them at runtime via runtime.GOARCH. Use a bind mount so only the tree matching
# $TARGETARCH ends up in an image layer (the unused arch never materialises).
ARG TARGETARCH
RUN --mount=type=bind,source=assets/tools,target=/ctx/tools,readonly \
    mkdir -p /app/assets/tools && \
    if [ "$TARGETARCH" = "amd64" ]; then \
      cp -r /ctx/tools/x64 /app/assets/tools/x64; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
      cp -r /ctx/tools/arm /app/assets/tools/arm; \
    fi && \
    chmod +x /app/assets/tools/*/postgresql/*/bin/* \
             /app/assets/tools/*/mysql/*/bin/* \
             /app/assets/tools/*/mariadb/*/bin/* \
             /app/assets/tools/*/mongodb/bin/*

# Create postgres user and set up directories
RUN groupadd -g 999 postgres || true && \
  useradd -m -s /bin/bash -u 999 -g 999 postgres || true && \
  mkdir -p /databasus-data/pgdata && \
  chown -R postgres:postgres /databasus-data/pgdata

# Create non-root user for the main application process
RUN useradd -r -s /usr/sbin/nologin -u 65532 databasus

WORKDIR /app

# Copy Goose from build stage
COPY --from=backend-build /usr/local/bin/goose /usr/local/bin/goose

# Copy Caddy for optional self-signed HTTPS termination
COPY --from=caddy /usr/bin/caddy /usr/bin/caddy
COPY caddy/Caddyfile /app/Caddyfile
COPY caddy/Caddyfile.custom-cert /app/Caddyfile.custom-cert
COPY caddy/start-caddy.sh /app/start-caddy.sh

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
COPY --from=verification-agent-build /verification-agent-binaries/* ./agent-binaries/

# Bake .env.example as /.env so the binary has defaults when no env file is
# mounted. The backend looks for .env at the parent of cwd (= /app), i.e. /.
# Real env vars (-e, compose, k8s) take precedence — godotenv.Load does not
# overwrite already-set variables.
COPY .env.example /.env

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
  IS_DISABLE_CLOUD_NOTICE: '\${IS_DISABLE_CLOUD_NOTICE:-false}',
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

/app/start-caddy.sh

exec gosu databasus ./main
EOF

LABEL org.opencontainers.image.source="https://github.com/databasus/databasus"

RUN chmod +x /app/start.sh /app/start-caddy.sh

EXPOSE 4005 4006

# Volume for PostgreSQL data
VOLUME ["/databasus-data"]

ENTRYPOINT ["/app/start.sh"]
CMD []

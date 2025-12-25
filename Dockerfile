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
FROM --platform=$BUILDPLATFORM golang:1.24.4 AS backend-build

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


# ========= RUNTIME =========
FROM debian:bookworm-slim

# Add version metadata to runtime image
ARG APP_VERSION=dev
LABEL org.opencontainers.image.version=$APP_VERSION
ENV APP_VERSION=$APP_VERSION

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
# Use dpkg with apt-get -f install to handle dependencies
RUN apt-get update && \
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-x86_64-100.10.0.deb -O /tmp/mongodb-database-tools.deb; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  wget -q https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-aarch64-100.10.0.deb -O /tmp/mongodb-database-tools.deb; \
  fi && \
  dpkg -i /tmp/mongodb-database-tools.deb || true && \
  apt-get install -f -y --no-install-recommends && \
  rm /tmp/mongodb-database-tools.deb && \
  rm -rf /var/lib/apt/lists/* && \
  ln -sf /usr/bin/mongodump /usr/local/mongodb-database-tools/bin/mongodump && \
  ln -sf /usr/bin/mongorestore /usr/local/mongodb-database-tools/bin/mongorestore

# Create postgres user and set up directories
RUN useradd -m -s /bin/bash postgres || true && \
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

# PostgreSQL 17 binary paths
PG_BIN="/usr/lib/postgresql/17/bin"

# Ensure proper ownership of data directory
echo "Setting up data directory permissions..."
mkdir -p /databasus-data/pgdata
chown -R postgres:postgres /databasus-data

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

# Start PostgreSQL in background
echo "Starting PostgreSQL..."
gosu postgres \$PG_BIN/postgres -D /databasus-data/pgdata -p 5437 &
POSTGRES_PID=\$!

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if gosu postgres \$PG_BIN/pg_isready -p 5437 -h localhost >/dev/null 2>&1; then
        echo "PostgreSQL is ready!"
        break
    fi
    if [ \$i -eq 30 ]; then
        echo "PostgreSQL failed to start"
        exit 1
    fi
    sleep 1
done

# Create database and set password for postgres user
echo "Setting up database and user..."
gosu postgres \$PG_BIN/psql -p 5437 -h localhost -d postgres << 'SQL'
ALTER USER postgres WITH PASSWORD 'Q1234567';
SELECT 'CREATE DATABASE databasus OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'databasus')
\\gexec
\\q
SQL

# Start the main application
echo "Starting Databasus application..."
exec ./main
EOF

RUN chmod +x /app/start.sh

EXPOSE 4005

# Volume for PostgreSQL data
VOLUME ["/databasus-data"]

ENTRYPOINT ["/app/start.sh"]
CMD []
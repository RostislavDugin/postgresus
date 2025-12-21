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

# Install PostgreSQL server and client tools (versions 12-18), MySQL client tools (5.7, 8.0, 8.4), MariaDB client tools, and rclone
# Note: MySQL 5.7 is only available for x86_64, MySQL 8.0+ supports both x86_64 and ARM64
# Note: MySQL binaries require libncurses5 for terminal handling
# Note: MariaDB uses a single client version (12.1) that is backward compatible with all server versions
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends \
  wget ca-certificates gnupg lsb-release sudo gosu curl unzip xz-utils libncurses5 && \
  # Add PostgreSQL repository
  wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - && \
  echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
  > /etc/apt/sources.list.d/pgdg.list && \
  apt-get update && \
  # Install PostgreSQL
  apt-get install -y --no-install-recommends \
  postgresql-17 postgresql-18 postgresql-client-12 postgresql-client-13 postgresql-client-14 postgresql-client-15 \
  postgresql-client-16 postgresql-client-17 postgresql-client-18 rclone && \
  # Create MySQL directories
  mkdir -p /usr/local/mysql-5.7/bin /usr/local/mysql-8.0/bin /usr/local/mysql-8.4/bin && \
  # Download and install MySQL client tools (architecture-aware)
  # MySQL 5.7: Only available for x86_64
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://dev.mysql.com/get/Downloads/MySQL-5.7/mysql-5.7.44-linux-glibc2.12-x86_64.tar.gz -O /tmp/mysql57.tar.gz && \
  tar -xzf /tmp/mysql57.tar.gz -C /tmp && \
  cp /tmp/mysql-5.7.*/bin/mysql /usr/local/mysql-5.7/bin/ && \
  cp /tmp/mysql-5.7.*/bin/mysqldump /usr/local/mysql-5.7/bin/ && \
  rm -rf /tmp/mysql-5.7.* /tmp/mysql57.tar.gz; \
  else \
  echo "MySQL 5.7 not available for $TARGETARCH, skipping..."; \
  fi && \
  # MySQL 8.0: Available for both x86_64 and ARM64
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-8.0.40-linux-glibc2.17-x86_64-minimal.tar.xz -O /tmp/mysql80.tar.xz; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  wget -q https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-8.0.40-linux-glibc2.17-aarch64-minimal.tar.xz -O /tmp/mysql80.tar.xz; \
  fi && \
  tar -xJf /tmp/mysql80.tar.xz -C /tmp && \
  cp /tmp/mysql-8.0.*/bin/mysql /usr/local/mysql-8.0/bin/ && \
  cp /tmp/mysql-8.0.*/bin/mysqldump /usr/local/mysql-8.0/bin/ && \
  rm -rf /tmp/mysql-8.0.* /tmp/mysql80.tar.xz && \
  # MySQL 8.4: Available for both x86_64 and ARM64
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://dev.mysql.com/get/Downloads/MySQL-8.4/mysql-8.4.3-linux-glibc2.17-x86_64-minimal.tar.xz -O /tmp/mysql84.tar.xz; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  wget -q https://dev.mysql.com/get/Downloads/MySQL-8.4/mysql-8.4.3-linux-glibc2.17-aarch64-minimal.tar.xz -O /tmp/mysql84.tar.xz; \
  fi && \
  tar -xJf /tmp/mysql84.tar.xz -C /tmp && \
  cp /tmp/mysql-8.4.*/bin/mysql /usr/local/mysql-8.4/bin/ && \
  cp /tmp/mysql-8.4.*/bin/mysqldump /usr/local/mysql-8.4/bin/ && \
  rm -rf /tmp/mysql-8.4.* /tmp/mysql84.tar.xz && \
  # Make MySQL binaries executable (ignore errors for empty dirs on ARM64)
  chmod +x /usr/local/mysql-*/bin/* 2>/dev/null || true && \
  # Create MariaDB directories for both versions
  # MariaDB uses two client versions:
  # - 10.6 (legacy): For older servers (5.5, 10.1) that don't have generation_expression column
  # - 12.1 (modern): For newer servers (10.2+)
  mkdir -p /usr/local/mariadb-10.6/bin /usr/local/mariadb-12.1/bin && \
  # Download and install MariaDB 10.6 client tools (legacy - for older servers)
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://archive.mariadb.org/mariadb-10.6.21/bintar-linux-systemd-x86_64/mariadb-10.6.21-linux-systemd-x86_64.tar.gz -O /tmp/mariadb106.tar.gz && \
  tar -xzf /tmp/mariadb106.tar.gz -C /tmp && \
  cp /tmp/mariadb-10.6.*/bin/mariadb /usr/local/mariadb-10.6/bin/ && \
  cp /tmp/mariadb-10.6.*/bin/mariadb-dump /usr/local/mariadb-10.6/bin/ && \
  rm -rf /tmp/mariadb-10.6.* /tmp/mariadb106.tar.gz; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  # For ARM64, install MariaDB 10.6 client from official repository
  curl -fsSL https://mariadb.org/mariadb_release_signing_key.asc | gpg --dearmor -o /usr/share/keyrings/mariadb-keyring.gpg && \
  echo "deb [signed-by=/usr/share/keyrings/mariadb-keyring.gpg] https://mirror.mariadb.org/repo/10.6/debian $(lsb_release -cs) main" > /etc/apt/sources.list.d/mariadb106.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends mariadb-client && \
  cp /usr/bin/mariadb /usr/local/mariadb-10.6/bin/mariadb && \
  cp /usr/bin/mariadb-dump /usr/local/mariadb-10.6/bin/mariadb-dump && \
  apt-get remove -y mariadb-client && \
  rm /etc/apt/sources.list.d/mariadb106.list; \
  fi && \
  # Download and install MariaDB 12.1 client tools (modern - for newer servers)
  if [ "$TARGETARCH" = "amd64" ]; then \
  wget -q https://archive.mariadb.org/mariadb-12.1.2/bintar-linux-systemd-x86_64/mariadb-12.1.2-linux-systemd-x86_64.tar.gz -O /tmp/mariadb121.tar.gz && \
  tar -xzf /tmp/mariadb121.tar.gz -C /tmp && \
  cp /tmp/mariadb-12.1.*/bin/mariadb /usr/local/mariadb-12.1/bin/ && \
  cp /tmp/mariadb-12.1.*/bin/mariadb-dump /usr/local/mariadb-12.1/bin/ && \
  rm -rf /tmp/mariadb-12.1.* /tmp/mariadb121.tar.gz; \
  elif [ "$TARGETARCH" = "arm64" ]; then \
  # For ARM64, install MariaDB 12.1 client from official repository
  echo "deb [signed-by=/usr/share/keyrings/mariadb-keyring.gpg] https://mirror.mariadb.org/repo/12.1/debian $(lsb_release -cs) main" > /etc/apt/sources.list.d/mariadb121.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends mariadb-client && \
  cp /usr/bin/mariadb /usr/local/mariadb-12.1/bin/mariadb && \
  cp /usr/bin/mariadb-dump /usr/local/mariadb-12.1/bin/mariadb-dump; \
  fi && \
  # Make MariaDB binaries executable
  chmod +x /usr/local/mariadb-*/bin/* 2>/dev/null || true && \
  # Cleanup
  rm -rf /var/lib/apt/lists/*

# Create postgres user and set up directories
RUN useradd -m -s /bin/bash postgres || true && \
  mkdir -p /postgresus-data/pgdata && \
  chown -R postgres:postgres /postgresus-data/pgdata

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

# PostgreSQL 17 binary paths
PG_BIN="/usr/lib/postgresql/17/bin"

# Ensure proper ownership of data directory
echo "Setting up data directory permissions..."
mkdir -p /postgresus-data/pgdata
chown -R postgres:postgres /postgresus-data

# Initialize PostgreSQL if not already initialized
if [ ! -s "/postgresus-data/pgdata/PG_VERSION" ]; then
    echo "Initializing PostgreSQL database..."
    gosu postgres \$PG_BIN/initdb -D /postgresus-data/pgdata --encoding=UTF8 --locale=C.UTF-8
    
    # Configure PostgreSQL
    echo "host all all 127.0.0.1/32 md5" >> /postgresus-data/pgdata/pg_hba.conf
    echo "local all all trust" >> /postgresus-data/pgdata/pg_hba.conf
    echo "port = 5437" >> /postgresus-data/pgdata/postgresql.conf
    echo "listen_addresses = 'localhost'" >> /postgresus-data/pgdata/postgresql.conf
    echo "shared_buffers = 256MB" >> /postgresus-data/pgdata/postgresql.conf
    echo "max_connections = 100" >> /postgresus-data/pgdata/postgresql.conf
fi

# Start PostgreSQL in background
echo "Starting PostgreSQL..."
gosu postgres \$PG_BIN/postgres -D /postgresus-data/pgdata -p 5437 &
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
SELECT 'CREATE DATABASE postgresus OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'postgresus')
\\gexec
\\q
SQL

# Start the main application
echo "Starting Postgresus application..."
exec ./main
EOF

RUN chmod +x /app/start.sh

EXPOSE 4005

# Volume for PostgreSQL data
VOLUME ["/postgresus-data"]

ENTRYPOINT ["/app/start.sh"]
CMD []
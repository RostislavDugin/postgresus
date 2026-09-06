#!/bin/bash
set -e

export TMPDIR=/tmp

readonly permissions_documentation_url="https://databasus.com/advanced-config/#docker-storage-permissions"
readonly postgres_binary_directory="/usr/lib/postgresql/17/bin"

runtime_uid=""
runtime_gid=""
postgres_pid=""
internal_postgres_password=""

reject_legacy_postgresus_volume() {
    if [ ! -d /postgresus-data ] || [ -z "$(ls -A /postgresus-data 2>/dev/null)" ]; then
        return
    fi

    echo ""
    echo "=========================================="
    echo "ERROR: Legacy volume detected!"
    echo "=========================================="
    echo ""
    echo "You are using the /postgresus-data folder. It seems you changed the image name from Postgresus to Databasus without changing the volume."
    echo ""
    echo "Please either:"
    echo "  1. Switch back to image rostislavdugin/postgresus:latest (supported until ~Dec 2026)"
    echo "  2. Read the migration guide: https://databasus.com/installation/#postgresus-migration"
    echo ""
    echo "=========================================="
    exit 1
}

report_identity_error() {
    local variable_name="$1"
    local variable_value="$2"

    echo "ERROR: ${variable_name} must be a non-zero decimal Linux ID no greater than 4294967294. Received: '${variable_value}'." >&2
    echo "See ${permissions_documentation_url}" >&2
    exit 1
}

validate_linux_id() {
    local variable_name="$1"
    local variable_value="$2"
    local maximum_linux_id=4294967294
    local normalized_id

    if [[ ! "${variable_value}" =~ ^[0-9]+$ ]]; then
        report_identity_error "${variable_name}" "${variable_value}"
    fi

    normalized_id="${variable_value#"${variable_value%%[!0]*}"}"
    normalized_id="${normalized_id:-0}"
    if [ "${normalized_id}" = "0" ] ||
       ((${#normalized_id} > ${#maximum_linux_id})) ||
       { ((${#normalized_id} == ${#maximum_linux_id})) && ((10#${normalized_id} > maximum_linux_id)); }; then
        report_identity_error "${variable_name}" "${variable_value}"
    fi

    echo "${normalized_id}"
}

get_mounted_id() {
    local stat_format="$1"
    local mounted_path
    local mounted_id
    local -a mounted_paths=()

    if [ -d /databasus-data/pgdata ]; then
        mounted_paths+=(/databasus-data/pgdata)
    fi
    mounted_paths+=(/databasus-data/backups /databasus-data)

    for mounted_path in "${mounted_paths[@]}"; do
        if [ ! -e "${mounted_path}" ]; then
            continue
        fi

        mounted_id="$(stat -c "${stat_format}" "${mounted_path}")"
        if [ "${mounted_id}" != "0" ]; then
            echo "${mounted_id}"
            return
        fi
    done

    echo 999
}

resolve_runtime_id() {
    local variable_name="$1"
    local stat_format="$2"
    local variable_value

    if [ "${!variable_name+x}" = "x" ]; then
        variable_value="${!variable_name}"
        validate_linux_id "${variable_name}" "${variable_value}"
        return
    fi

    get_mounted_id "${stat_format}"
}

ensure_id_is_available() {
    local account_database="$1"
    local identity_kind="$2"
    local selected_id="$3"
    local owning_name

    owning_name="$(getent "${account_database}" "${selected_id}" | cut -d: -f1 || true)"
    if [ -z "${owning_name}" ] || [ "${owning_name}" = "databasus" ]; then
        return
    fi

    echo "ERROR: Selected ${identity_kind} ${selected_id} belongs to account '${owning_name}', not 'databasus'." >&2
    echo "Set PUID and PGID to unused IDs or fix the mounted directory ownership:" >&2
    echo "${permissions_documentation_url}" >&2
    exit 1
}

configure_runtime_identity() {
    runtime_uid="$(resolve_runtime_id PUID %u)"
    runtime_gid="$(resolve_runtime_id PGID %g)"

    ensure_id_is_available passwd UID "${runtime_uid}"
    ensure_id_is_available group GID "${runtime_gid}"

    if [ "$(id -g databasus)" != "${runtime_gid}" ]; then
        groupmod -g "${runtime_gid}" databasus
    fi
    if [ "$(id -u databasus)" != "${runtime_uid}" ]; then
        usermod -u "${runtime_uid}" databasus
    fi

    echo "Using databasus runtime identity ${runtime_uid}:${runtime_gid}."
}

generate_frontend_runtime_configuration() {
    local is_email_configured=false

    if [ -n "${SMTP_HOST:-}" ] && [ -n "${DATABASUS_URL:-}" ]; then
        is_email_configured=true
    fi

    echo "Generating runtime configuration..."
    cat > /app/ui/build/runtime-config.js <<JSEOF
// Runtime configuration injected at container startup
// This file is generated dynamically and should not be edited manually
window.__RUNTIME_CONFIG__ = {
  GITHUB_CLIENT_ID: '${GITHUB_CLIENT_ID:-}',
  GOOGLE_CLIENT_ID: '${GOOGLE_CLIENT_ID:-}',
  IS_EMAIL_CONFIGURED: '${is_email_configured}',
  CLOUDFLARE_TURNSTILE_SITE_KEY: '${CLOUDFLARE_TURNSTILE_SITE_KEY:-}',
  CONTAINER_ARCH: '${CONTAINER_ARCH:-unknown}'
};
JSEOF

    if [ -n "${ANALYTICS_SCRIPT:-}" ] &&
       ! grep -q "rybbit.databasus.com" /app/ui/build/index.html 2>/dev/null; then
        echo "Injecting analytics script..."
        sed -i "s#</head>#  ${ANALYTICS_SCRIPT}\
  </head>#" /app/ui/build/index.html
    fi
}

report_storage_error() {
    local storage_path="$1"
    local required_operation="$2"

    echo "ERROR: Databasus cannot write to ${storage_path} as UID ${runtime_uid} and GID ${runtime_gid}." >&2
    echo "Required operation: ${required_operation}." >&2
    echo "Set PUID and PGID or fix the mounted directory permissions:" >&2
    echo "${permissions_documentation_url}" >&2
    return 1
}

create_required_data_directories() {
    local required_directory

    for required_directory in \
        /databasus-data \
        /databasus-data/pgdata \
        /databasus-data/pgsocket \
        /databasus-data/temp \
        /databasus-data/backups; do
        if mkdir -p "${required_directory}" 2>/dev/null ||
           gosu databasus mkdir -p "${required_directory}"; then
            continue
        fi

        report_storage_error "${required_directory}" "create the directory"
        return 1
    done
}

normalize_data_permissions() {
    chown "${runtime_uid}:${runtime_gid}" /databasus-data 2>/dev/null || true
    chmod 0770 /databasus-data 2>/dev/null || true
    chown "${runtime_uid}:${runtime_gid}" \
        /databasus-data/pgdata \
        /databasus-data/pgsocket \
        /databasus-data/temp \
        /databasus-data/backups 2>/dev/null || true
    chmod 0700 \
        /databasus-data/pgdata \
        /databasus-data/pgsocket \
        /databasus-data/temp 2>/dev/null || true
    chmod 0770 /databasus-data/backups 2>/dev/null || true
}

can_read_and_open_file_for_writing() {
    local checked_file="$1"

    gosu databasus test -r "${checked_file}" &&
        gosu databasus tee -a "${checked_file}" </dev/null >/dev/null
}

verify_existing_data_files() {
    local existing_data_file

    for existing_data_file in \
        /databasus-data/secret.key \
        /databasus-data/instance.json \
        /databasus-data/databasus.log; do
        if [ ! -e "${existing_data_file}" ]; then
            continue
        fi

        chown "${runtime_uid}:${runtime_gid}" "${existing_data_file}" 2>/dev/null || true
        chmod 0600 "${existing_data_file}" 2>/dev/null || true

        if can_read_and_open_file_for_writing "${existing_data_file}"; then
            continue
        fi

        report_storage_error "${existing_data_file}" "read and open the existing file for writing"
        return 1
    done
}

verify_directory_file_lifecycle() {
    local checked_directory="$1"
    local probe_path="${checked_directory}/.databasus-required-write-check-$$-$RANDOM"
    local probe_content="$2"
    local persisted_probe_content

    if ! printf "%s" "${probe_content}" |
        gosu databasus tee "${probe_path}" >/dev/null; then
        gosu databasus rm -f "${probe_path}" 2>/dev/null || true
        return 1
    fi

    persisted_probe_content="$(gosu databasus cat "${probe_path}" 2>/dev/null || true)"
    if [ "${persisted_probe_content}" != "${probe_content}" ]; then
        gosu databasus rm -f "${probe_path}" 2>/dev/null || true
        return 1
    fi

    gosu databasus rm "${probe_path}"
}

verify_wal_queue_access() {
    local wal_queue_directory=/databasus-data/backups/wal-queue
    local queue_listing
    local queue_directory
    local queue_file
    local inaccessible_queue_directory=""
    local inaccessible_queue_file=""

    if [ ! -d "${wal_queue_directory}" ]; then
        return
    fi

    chown -R "${runtime_uid}:${runtime_gid}" "${wal_queue_directory}" 2>/dev/null || true
    chmod -R u+rwX "${wal_queue_directory}" 2>/dev/null || true

    queue_listing="$(mktemp)"
    if ! gosu databasus find "${wal_queue_directory}" -type d -print0 > "${queue_listing}"; then
        rm -f "${queue_listing}"
        report_storage_error "${wal_queue_directory}" "traverse the existing WAL queue"
        return 1
    fi

    while IFS= read -r -d "" queue_directory; do
        if verify_directory_file_lifecycle "${queue_directory}" queue-check; then
            continue
        fi

        inaccessible_queue_directory="${queue_directory}"
        break
    done < "${queue_listing}"
    if [ -n "${inaccessible_queue_directory}" ]; then
        rm -f "${queue_listing}"
        report_storage_error "${inaccessible_queue_directory}" "create, read, and remove a WAL queue file"
        return 1
    fi

    if ! gosu databasus find "${wal_queue_directory}" -type f -print0 > "${queue_listing}"; then
        rm -f "${queue_listing}"
        report_storage_error "${wal_queue_directory}" "traverse the existing WAL queue"
        return 1
    fi

    while IFS= read -r -d "" queue_file; do
        if can_read_and_open_file_for_writing "${queue_file}"; then
            continue
        fi

        inaccessible_queue_file="${queue_file}"
        break
    done < "${queue_listing}"
    if [ -n "${inaccessible_queue_file}" ]; then
        rm -f "${queue_listing}"
        report_storage_error "${inaccessible_queue_file}" "read and open the existing WAL queue file for writing"
        return 1
    fi

    rm -f "${queue_listing}"
}

verify_service_directories() {
    local required_write_directory

    for required_write_directory in \
        /databasus-data \
        /databasus-data/pgdata \
        /databasus-data/pgsocket; do
        if verify_directory_file_lifecycle "${required_write_directory}" service-check; then
            continue
        fi

        report_storage_error "${required_write_directory}" "create, read, and remove a required file"
        return 1
    done
}

prepare_and_verify_storage() {
    echo "Setting up data directory permissions..."

    create_required_data_directories
    normalize_data_permissions
    verify_existing_data_files
    verify_wal_queue_access
    verify_service_directories
    gosu databasus /app/main --test-storage
}

configure_and_start_valkey() {
    echo "Configuring Valkey cache..."
    cat > /tmp/valkey.conf <<'VALKEY_CONFIG'
port 6379
bind 127.0.0.1
protected-mode yes
save ""
maxmemory 256mb
maxmemory-policy allkeys-lru
VALKEY_CONFIG

    echo "Starting Valkey..."
    gosu databasus valkey-server /tmp/valkey.conf &

    echo "Waiting for Valkey to be ready..."
    local valkey_readiness_attempt
    for ((valkey_readiness_attempt = 1; valkey_readiness_attempt <= 30; valkey_readiness_attempt++)); do
        if gosu databasus valkey-cli ping >/dev/null 2>&1; then
            echo "Valkey is ready!"
            return
        fi
        sleep 1
    done

    echo "ERROR: Valkey did not become ready." >&2
    return 1
}

initialize_postgresql_cluster() {
    if [ -s /databasus-data/pgdata/PG_VERSION ]; then
        return
    fi

    echo "Initializing PostgreSQL database..."
    gosu databasus "${postgres_binary_directory}/initdb" \
        -D /databasus-data/pgdata \
        --encoding=UTF8 \
        --locale=C.UTF-8 \
        --username=postgres

    {
        echo "port = 5437"
        echo "listen_addresses = 'localhost'"
        echo "shared_buffers = 256MB"
        echo "max_connections = 100"
    } >> /databasus-data/pgdata/postgresql.conf
}

write_bootstrap_postgresql_authentication() {
    gosu databasus tee /databasus-data/pgdata/pg_ident.conf.new >/dev/null <<'PG_IDENT'
databasus_bootstrap databasus postgres
PG_IDENT
    gosu databasus chmod 0600 /databasus-data/pgdata/pg_ident.conf.new
    gosu databasus mv \
        /databasus-data/pgdata/pg_ident.conf.new \
        /databasus-data/pgdata/pg_ident.conf

    gosu databasus tee /databasus-data/pgdata/pg_hba.conf.new >/dev/null <<'PG_HBA'
local all postgres peer map=databasus_bootstrap
local all all reject
local replication all reject
host all all 127.0.0.1/32 scram-sha-256
host all all ::1/128 scram-sha-256
host replication all 127.0.0.1/32 reject
host replication all ::1/128 reject
PG_HBA
    gosu databasus chmod 0600 /databasus-data/pgdata/pg_hba.conf.new
    gosu databasus mv \
        /databasus-data/pgdata/pg_hba.conf.new \
        /databasus-data/pgdata/pg_hba.conf
}

start_postgresql() {
    echo "Starting PostgreSQL..."
    gosu databasus "${postgres_binary_directory}/postgres" \
        -D /databasus-data/pgdata \
        -p 5437 \
        -k /databasus-data/pgsocket \
        -c password_encryption=scram-sha-256 &
    postgres_pid=$!

    echo "Waiting for PostgreSQL to be ready..."
    local postgres_readiness_attempt
    for ((postgres_readiness_attempt = 1; postgres_readiness_attempt <= 30; postgres_readiness_attempt++)); do
        if gosu databasus "${postgres_binary_directory}/pg_isready" \
            -p 5437 -h /databasus-data/pgsocket >/dev/null 2>&1; then
            echo "PostgreSQL is ready!"
            return
        fi
        sleep 1
    done

    return 1
}

report_postgresql_recovery_failure() {
    echo ""
    echo "=========================================="
    echo "ERROR: PostgreSQL failed to start even after WAL reset."
    echo "The database may be severely corrupted."
    echo ""
    echo "Options:"
    echo "  1. Delete the volume and start fresh (data loss)"
    echo "  2. Manually inspect /databasus-data/pgdata for issues"
    echo "=========================================="
    return 1
}

start_postgresql_with_recovery() {
    if start_postgresql; then
        return
    fi

    echo ""
    echo "=========================================="
    echo "PostgreSQL failed to start. Attempting WAL reset recovery..."
    echo "=========================================="
    echo ""

    kill -9 "${postgres_pid}" 2>/dev/null || true
    sleep 2

    echo "Running pg_resetwal to reset WAL..."
    if ! gosu databasus "${postgres_binary_directory}/pg_resetwal" \
        -f /databasus-data/pgdata; then
        echo ""
        echo "=========================================="
        echo "ERROR: pg_resetwal failed."
        echo "The database may be severely corrupted."
        echo ""
        echo "Options:"
        echo "  1. Delete the volume and start fresh (data loss)"
        echo "  2. Manually inspect /databasus-data/pgdata for issues"
        echo "=========================================="
        return 1
    fi

    echo "WAL reset successful. Restarting PostgreSQL..."
    if ! start_postgresql; then
        report_postgresql_recovery_failure
        return 1
    fi

    echo "PostgreSQL recovered successfully after WAL reset!"
}

configure_postgresql_database() {
    internal_postgres_password="$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')"

    echo "Setting up database and user..."
    gosu databasus "${postgres_binary_directory}/psql" \
        -v ON_ERROR_STOP=1 \
        -v internal_postgres_password="${internal_postgres_password}" \
        -p 5437 \
        -h /databasus-data/pgsocket \
        -U postgres \
        -d postgres <<'SQL'

ALTER USER postgres WITH PASSWORD :'internal_postgres_password';
SELECT 'CREATE DATABASE databasus OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'databasus')
\gexec
\q
SQL
}

activate_runtime_postgresql_authentication() {
    gosu databasus tee /databasus-data/pgdata/pg_hba.conf.new >/dev/null <<'PG_HBA'
local all all reject
local replication all reject
host all all 127.0.0.1/32 scram-sha-256
host all all ::1/128 scram-sha-256
host replication all 127.0.0.1/32 reject
host replication all ::1/128 reject
PG_HBA
    gosu databasus chmod 0600 /databasus-data/pgdata/pg_hba.conf.new
    gosu databasus mv \
        /databasus-data/pgdata/pg_hba.conf.new \
        /databasus-data/pgdata/pg_hba.conf
    gosu databasus /bin/sh -c ': > /databasus-data/pgdata/pg_ident.conf'
    gosu databasus chmod 0600 /databasus-data/pgdata/pg_ident.conf
    gosu databasus "${postgres_binary_directory}/pg_ctl" \
        -D /databasus-data/pgdata reload
}

verify_postgresql_runtime_authentication() {
    if gosu databasus "${postgres_binary_directory}/psql" \
        -p 5437 \
        -h /databasus-data/pgsocket \
        -U postgres \
        -d postgres \
        -c 'SELECT 1' >/dev/null 2>&1; then
        echo "ERROR: Embedded PostgreSQL still accepts Unix-socket login after bootstrap." >&2
        return 1
    fi

    if ! gosu databasus env PGPASSWORD="${internal_postgres_password}" \
        "${postgres_binary_directory}/psql" \
        -h localhost \
        -p 5437 \
        -U postgres \
        -d postgres \
        -c 'SELECT 1' >/dev/null; then
        echo "ERROR: Embedded PostgreSQL rejected its generated loopback credential." >&2
        return 1
    fi
}

configure_application_database_dsn() {
    if [ -n "${DATABASE_DSN+x}" ]; then
        return
    fi

    export DATABASE_DSN="host=localhost user=postgres password=${internal_postgres_password} dbname=databasus port=5437 sslmode=disable"
}

bootstrap_postgresql() {
    initialize_postgresql_cluster
    write_bootstrap_postgresql_authentication
    start_postgresql_with_recovery
    configure_postgresql_database
    activate_runtime_postgresql_authentication
    verify_postgresql_runtime_authentication
    configure_application_database_dsn
}

reject_legacy_wal_configuration() {
    local has_backup_type_column
    local has_legacy_wal_database

    echo "Checking for legacy WAL backup configuration..."
    has_backup_type_column="$(gosu databasus env PGPASSWORD="${internal_postgres_password}" \
        "${postgres_binary_directory}/psql" \
        -h localhost -p 5437 -U postgres -d databasus -tA \
        -c "SELECT 1 FROM information_schema.columns WHERE table_name='postgresql_databases' AND column_name='backup_type' LIMIT 1" \
        2>/dev/null || true)"

    if [ "${has_backup_type_column}" != "1" ]; then
        echo "No legacy WAL backup data detected."
        return
    fi

    has_legacy_wal_database="$(gosu databasus env PGPASSWORD="${internal_postgres_password}" \
        "${postgres_binary_directory}/psql" \
        -h localhost -p 5437 -U postgres -d databasus -tA \
        -c "SELECT 1 FROM postgresql_databases WHERE backup_type='WAL_V1' LIMIT 1" \
        2>/dev/null || true)"

    if [ "${has_legacy_wal_database}" != "1" ]; then
        echo "No legacy WAL backup data detected."
        return
    fi

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
    return 1
}

main() {
    reject_legacy_postgresus_volume
    configure_runtime_identity
    generate_frontend_runtime_configuration
    configure_and_start_valkey
    prepare_and_verify_storage
    bootstrap_postgresql
    reject_legacy_wal_configuration

    echo "Starting Databasus application..."
    exec gosu databasus ./main
}

main "$@"

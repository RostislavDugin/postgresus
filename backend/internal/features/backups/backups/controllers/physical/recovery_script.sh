#!/bin/sh
# Databasus physical-restore helper.
#
# Takes a restore bundle produced by the Databasus restore stream, decompresses
# what needs decompressing, runs pg_combinebackup over the full + incremental
# chain, and wires up WAL replay / point-in-time recovery.
#
# Usage:
#   sh databasus-recovery.sh [--pg-bin <dir>] [--combine-image <image>] [--target-time <utc-ts>] <bundle-url-or-path> [output-dir]
#
#   --pg-bin <dir>        directory holding the PostgreSQL tools (pg_combinebackup,
#                         pg_ctl) when they are not on PATH.
#   --combine-image <img> run pg_combinebackup inside this Docker image instead of on
#                         the host - for hosts that have zstd but no PostgreSQL tools.
#                         The host still decompresses the backup blobs and WAL with its
#                         zstd; only the version-specific pg_combinebackup runs in the
#                         container, and the started cluster needs no extra tools.
#   --target-time <ts>    point-in-time recovery target, a PostgreSQL-parseable UTC
#                         timestamp (e.g. '2026-06-12 14:30:00+00:00'). Replays WAL
#                         up to this point and promotes; omit to replay all shipped
#                         WAL (latest). Ignored when the bundle ships no WAL.
#   <bundle-url-or-path>  an http(s):// restore-stream URL (with its token) or a
#                         path to an already-downloaded .tar bundle.
#   [output-dir]          where to build the restore (default: databasus-restore).

set -eu

usage() {
    cat >&2 <<'USAGE'
usage: sh databasus-recovery.sh [--pg-bin <dir>] [--combine-image <image>] [--target-time <utc-ts>] <bundle-url-or-path> [output-dir]

  --pg-bin <dir>        directory holding the PostgreSQL tools (pg_combinebackup,
                        pg_ctl) when they are not on PATH
  --combine-image <img> run pg_combinebackup inside this Docker image instead of on
                        the host (decompression still uses the host's zstd)
  --target-time <ts>    point-in-time recovery target, a PostgreSQL-parseable UTC
                        timestamp (e.g. '2026-06-12 14:30:00+00:00'); omit to
                        replay all shipped WAL. Ignored when the bundle ships no WAL
  <bundle-url-or-path>  an http(s):// restore-stream URL (with its token) or a
                        path to an already-downloaded .tar bundle
  [output-dir]          where to build the restore (default: databasus-restore)
USAGE
}

PG_BIN=""
COMBINE_IMAGE=""
TARGET_TIME=""
while [ $# -gt 0 ]; do
    case "$1" in
        --pg-bin)
            [ $# -ge 2 ] || {
                echo "error: --pg-bin needs a directory" >&2
                exit 2
            }
            PG_BIN="$2"
            shift 2
            ;;
        --pg-bin=*)
            PG_BIN="${1#*=}"
            shift
            ;;
        --combine-image)
            [ $# -ge 2 ] || {
                echo "error: --combine-image needs an image reference" >&2
                exit 2
            }
            COMBINE_IMAGE="$2"
            shift 2
            ;;
        --combine-image=*)
            COMBINE_IMAGE="${1#*=}"
            shift
            ;;
        --target-time)
            [ $# -ge 2 ] || {
                echo "error: --target-time needs a timestamp" >&2
                exit 2
            }
            TARGET_TIME="$2"
            shift 2
            ;;
        --target-time=*)
            TARGET_TIME="${1#*=}"
            shift
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        -*)
            echo "error: unknown option '$1'" >&2
            usage
            exit 2
            ;;
        *)
            break
            ;;
    esac
done

SOURCE="${1:-}"
OUT_DIR="${2:-databasus-restore}"

if [ -z "$SOURCE" ]; then
    usage
    exit 2
fi

require_tool() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "error: '$1' is required but was not found on PATH" >&2
        [ -n "${2:-}" ] && echo "       $2" >&2
        exit 1
    fi
}

# The restore builds the cluster inside <out>. If <out> itself is already a live PostgreSQL
# data or install directory (PG_VERSION, pg_hba.conf, the binaries, ...), the user almost
# certainly aimed the restore at the wrong place: we would write the cluster a level deeper
# and the existing one would keep being served. Refuse up front rather than clobber or
# silently restore into the wrong layout.
require_empty_target() {
    target="$1"
    [ -d "$target" ] || return 0

    for marker in PG_VERSION postgresql.conf pg_hba.conf base global postmaster.pid \
        bin/postgres bin/pg_ctl; do
        if [ -e "$target/$marker" ]; then
            echo "error: output directory '$target' already looks like a PostgreSQL data or install directory (found '$marker')" >&2
            echo "       Point the restore at an empty directory instead." >&2
            exit 1
        fi
    done
}

# Major version out of a `... (PostgreSQL) 18.1` banner.
pg_major_from_banner() {
    echo "$1" | sed -n 's/.*PostgreSQL) \([0-9][0-9]*\).*/\1/p'
}

# Where the official postgres image expects PGDATA: PG 18 moved it under a version-specific
# path, PG <=17 keeps the flat one. The restored cluster must be bind-mounted here exactly.
container_pgdata() {
    if [ "${1:-0}" -ge 18 ] 2>/dev/null; then
        echo "/var/lib/postgresql/$1/docker"
    else
        echo "/var/lib/postgresql/data"
    fi
}

# Where to build the cluster under <out>, mirroring the image's PGDATA layout so a volume-root
# mount (-v <out>:/var/lib/postgresql on PG 18) finds it. PG 18 nests <major>/docker; PG <=17
# keeps <out>/data, which the user mounts straight at /var/lib/postgresql/data.
host_cluster_dir() {
    if [ "${2:-0}" -ge 18 ] 2>/dev/null; then
        echo "$1/$2/docker"
    else
        echo "$1/data"
    fi
}

# Dump pg_controldata for the reconstructed cluster in the C locale so its labels stay English
# (they are localized otherwise and the parse below would break). Runs the same way as
# pg_combinebackup: inside --combine-image, or off --pg-bin / PATH on the host.
control_data() {
    if [ -n "$COMBINE_IMAGE" ]; then
        docker run --rm --user "$(id -u):$(id -g)" -e LC_ALL=C -v "$OUT_ABS:$OUT_ABS" \
            "$COMBINE_IMAGE" pg_controldata "$DATA_DIR"
    else
        LC_ALL=C "${PG_BIN:+$PG_BIN/}pg_controldata" "$DATA_DIR"
    fi
}

# Read one "<label> setting:" value out of a captured pg_controldata dump.
control_setting() {
    echo "$2" | sed -n "s/^$1 setting: *//p" | tr -d '[:space:]'
}

# Resolve the PostgreSQL tools off --pg-bin when they are not on PATH; every later
# pg_combinebackup / pg_ctl lookup then finds them.
if [ -n "$PG_BIN" ]; then
    if [ ! -x "$PG_BIN/pg_combinebackup" ]; then
        echo "error: --pg-bin '$PG_BIN' has no executable pg_combinebackup" >&2
        exit 1
    fi

    PATH="$PG_BIN:$PATH"
    export PATH
fi

require_tool tar

# pg_combinebackup is version-specific. Either it is on the host (default) or it
# runs in --combine-image, in which case only Docker is required on the host.
if [ -n "$COMBINE_IMAGE" ]; then
    require_tool docker "needed to run pg_combinebackup inside '$COMBINE_IMAGE'"
else
    require_tool pg_combinebackup \
        "install the PostgreSQL client tools, pass --pg-bin <dir>, or use --combine-image <postgres-image>"
fi

require_empty_target "$OUT_DIR"

mkdir -p "$OUT_DIR"
OUT_ABS="$(cd "$OUT_DIR" && pwd)"
BUNDLE_DIR="$OUT_ABS/bundle"
mkdir -p "$BUNDLE_DIR"
# DATA_DIR is resolved once the backup's major version is known (it decides the layout).

# 1. Obtain the bundle.
case "$SOURCE" in
    http://* | https://*)
        TAR_PATH="$OUT_ABS/restore.tar"
        echo "Downloading restore bundle..."
        if command -v curl >/dev/null 2>&1; then
            curl -fSL "$SOURCE" -o "$TAR_PATH"
        elif command -v wget >/dev/null 2>&1; then
            wget -O "$TAR_PATH" "$SOURCE"
        else
            echo "error: need curl or wget to download $SOURCE" >&2
            exit 1
        fi
        ;;
    *)
        TAR_PATH="$SOURCE"
        if [ ! -f "$TAR_PATH" ]; then
            echo "error: bundle file not found: $TAR_PATH" >&2
            exit 1
        fi
        ;;
esac

# 2. Extract.
echo "Extracting bundle..."
tar -xf "$TAR_PATH" -C "$BUNDLE_DIR"

# 3. Verify the transfer before touching the bytes (compressed WAL included).
if [ -f "$BUNDLE_DIR/MANIFEST.sha256" ] && command -v sha256sum >/dev/null 2>&1; then
    echo "Verifying checksums..."
    (cd "$BUNDLE_DIR" && sha256sum -c MANIFEST.sha256 >/dev/null) ||
        {
            echo "error: checksum verification failed" >&2
            exit 1
        }
fi

# 4. Decompress each backup blob into its own directory, then fold the full +
#    incremental chain (oldest -> newest) with pg_combinebackup. Each backup ships as
#    <dir>/base.tar<ext> (still compressed) plus its <dir>/backup_manifest sidecar;
#    pg_combinebackup reads the manifest from the extracted directory.
echo "Reconstructing data directory with pg_combinebackup..."
INCR_DIRS="$(cd "$BUNDLE_DIR" && ls -d incr-* 2>/dev/null | sort -V || true)"
RECON_DIR="$BUNDLE_DIR/recon"
INPUTS=""
for backup in full $INCR_DIRS; do
    src="$BUNDLE_DIR/$backup"
    dst="$RECON_DIR/$backup"
    mkdir -p "$dst"

    blob="$(ls "$src"/base.tar* 2>/dev/null | head -n1 || true)"
    if [ -z "$blob" ]; then
        echo "error: backup '$backup' has no base.tar payload" >&2
        exit 1
    fi

    case "$blob" in
        *.zst)
            require_tool zstd \
                "the backup is zstd-compressed; install zstd (e.g. apt-get install zstd / brew install zstd) and re-run"
            zstd -dqc "$blob" | tar -xf - -C "$dst"
            ;;
        *.gz)
            gzip -dc "$blob" | tar -xf - -C "$dst"
            ;;
        *)
            tar -xf "$blob" -C "$dst"
            ;;
    esac

    cp "$src/backup_manifest" "$dst/backup_manifest"
    INPUTS="$INPUTS $dst"
done

# The backup's major version is its PG_VERSION (host-side file, so this works in --combine-image
# mode too). It decides where the cluster goes under <out>, mirroring the image's PGDATA layout.
RESTORED_MAJOR="$(tr -d '[:space:]' <"$RECON_DIR/full/PG_VERSION" 2>/dev/null || true)"
DATA_DIR="$(host_cluster_dir "$OUT_ABS" "$RESTORED_MAJOR")"

# Refuse to restore onto an already-initialized cluster at the target path - e.g. a volume the
# postgres image already booted once (PG 18 leaves it at <out>/<major>/docker). That is the exact
# empty-DB trap: we would write a second cluster the server never serves.
if [ -e "$DATA_DIR/PG_VERSION" ]; then
    echo "error: target cluster path '$DATA_DIR' already holds a PostgreSQL cluster" >&2
    echo "       This is usually a previous restore or a volume the postgres image already initialized." >&2
    echo "       Clear '$OUT_ABS' and re-run so the restored cluster is the only one there." >&2
    exit 1
fi

mkdir -p "$(dirname "$DATA_DIR")"
rm -rf "$DATA_DIR"
if [ -n "$COMBINE_IMAGE" ]; then
    # Run pg_combinebackup in the container as the host user (so the output stays
    # host-owned and the later chmod / WAL-wiring on the host can touch it) and bind
    # the output dir at its own absolute path so $INPUTS / $DATA_DIR resolve the same
    # inside and out.
    docker run --rm --user "$(id -u):$(id -g)" -v "$OUT_ABS:$OUT_ABS" -w "$OUT_ABS" \
        "$COMBINE_IMAGE" pg_combinebackup $INPUTS -o "$DATA_DIR"
else
    pg_combinebackup $INPUTS -o "$DATA_DIR"
fi
chmod 700 "$DATA_DIR"

# A data directory can only be started by a server of its own major version - the classic
# "restored 17, started 18" failure.
if [ -z "$COMBINE_IMAGE" ]; then
    # On the host, pg_ctl is the server that will run it, so we can check it now.
    TOOL_MAJOR="$(pg_major_from_banner "$(pg_ctl --version 2>/dev/null || true)")"
    if [ -n "$RESTORED_MAJOR" ] && [ -n "$TOOL_MAJOR" ] && [ "$RESTORED_MAJOR" != "$TOOL_MAJOR" ]; then
        echo "error: restored cluster is PostgreSQL $RESTORED_MAJOR but the PostgreSQL binaries are $TOOL_MAJOR" >&2
        echo "       A cluster can only be started by its own major version - install PostgreSQL $RESTORED_MAJOR (or pass --pg-bin <dir> pointing at it) and re-run." >&2
        exit 1
    fi
fi

# 5. Wire up WAL replay / PITR when the bundle ships WAL. A per-backup restore
#    ships none - its combined directory is already consistent, so we stop here.
WAL_DIR="$BUNDLE_DIR/wal"
if [ -d "$WAL_DIR" ] && [ -n "$(ls -A "$WAL_DIR" 2>/dev/null)" ]; then
    # Inflate every shipped WAL segment now, on the host, into a plaintext archive *inside the
    # data directory*. restore_command is then a plain `cp` from a path relative to PGDATA, so it
    # resolves whether the cluster is started on the host (pg_ctl -D) or in a container where
    # PGDATA is bind-mounted - the WAL travels with the cluster either way. It also never invokes
    # zstd (the host already decompressed it), so the postgres image needs no extra tools.
    PLAIN_WAL_DIR="$DATA_DIR/databasus_wal_restore"
    mkdir -p "$PLAIN_WAL_DIR"
    for seg in "$WAL_DIR"/*; do
        [ -e "$seg" ] || continue

        case "$seg" in
            *.zst)
                require_tool zstd \
                    "the shipped WAL is zstd-compressed; install zstd and re-run"
                zstd -dqc "$seg" >"$PLAIN_WAL_DIR/$(basename "${seg%.zst}")"
                ;;
            *)
                cp "$seg" "$PLAIN_WAL_DIR/"
                ;;
        esac
    done

    # PostgreSQL runs restore_command with the working directory set to PGDATA and %f/%p
    # substituted, so a path relative to PGDATA is portable across host and container starts. A
    # missing segment makes cp exit non-zero, which is the end-of-archive signal. Only double
    # quotes inside, so the value stays safely wrapped in single quotes.
    restore_command="cp \"databasus_wal_restore/%f\" \"%p\""

    {
        echo "archive_mode = off"
        echo "archive_command = ''"
        echo "restore_command = '$restore_command'"
        if [ -n "$TARGET_TIME" ]; then
            echo "recovery_target_time = '$TARGET_TIME'"
            echo "recovery_target_action = 'promote'"
        fi
    } >>"$DATA_DIR/postgresql.auto.conf"
    touch "$DATA_DIR/recovery.signal"

    # PostgreSQL aborts archive recovery if these five parameters are lower than they were on the
    # primary - they size shared-memory structures the WAL replay reconstructs. The primary's values
    # live in the control file, which is exactly what CheckRequiredParameterValues() compares against,
    # so pinning them >= those values lets recovery proceed without the user passing -c flags by hand.
    CONTROL_OUT="$(control_data 2>/dev/null || true)"
    MAX_CONNECTIONS="$(control_setting max_connections "$CONTROL_OUT")"
    MAX_WORKER_PROCESSES="$(control_setting max_worker_processes "$CONTROL_OUT")"
    MAX_WAL_SENDERS="$(control_setting max_wal_senders "$CONTROL_OUT")"
    MAX_PREPARED_XACTS="$(control_setting max_prepared_xacts "$CONTROL_OUT")"
    MAX_LOCKS_PER_XACT="$(control_setting max_locks_per_xact "$CONTROL_OUT")"

    if [ -n "$MAX_CONNECTIONS$MAX_WORKER_PROCESSES$MAX_WAL_SENDERS$MAX_PREPARED_XACTS$MAX_LOCKS_PER_XACT" ]; then
        {
            echo "# databasus restore: WAL replay needs these >= the primary's values (read from the"
            echo "# backup's control file via pg_controldata); safe to lower or remove after promotion."
            [ -n "$MAX_CONNECTIONS" ] && echo "max_connections = $MAX_CONNECTIONS"
            [ -n "$MAX_WORKER_PROCESSES" ] && echo "max_worker_processes = $MAX_WORKER_PROCESSES"
            [ -n "$MAX_WAL_SENDERS" ] && echo "max_wal_senders = $MAX_WAL_SENDERS"
            [ -n "$MAX_PREPARED_XACTS" ] && echo "max_prepared_transactions = $MAX_PREPARED_XACTS"
            [ -n "$MAX_LOCKS_PER_XACT" ] && echo "max_locks_per_transaction = $MAX_LOCKS_PER_XACT"
        } >>"$DATA_DIR/postgresql.auto.conf"
    else
        echo "warning: could not read the primary's parameters from the control file." >&2
        echo "         If recovery aborts with 'insufficient parameter settings', raise" >&2
        echo "         max_connections, max_worker_processes, max_wal_senders," >&2
        echo "         max_prepared_transactions and max_locks_per_transaction to at least" >&2
        echo "         the primary's values and restart." >&2
    fi
fi

CONTAINER_PGDATA="$(container_pgdata "$RESTORED_MAJOR")"

echo
echo "Restore prepared at: $DATA_DIR"
echo "Next steps:"
echo "  1. Ensure the directory is owned by the postgres OS user:"
echo "       chown -R postgres:postgres '$DATA_DIR'"
if [ -n "$COMBINE_IMAGE" ]; then
    echo "  2. Start PostgreSQL ${RESTORED_MAJOR:-<major>} with a matching postgres:${RESTORED_MAJOR:-<major>} image"
    echo "     (the image needs no extra tools - the host already decompressed the backup and WAL):"
    if [ "${RESTORED_MAJOR:-0}" -ge 18 ] 2>/dev/null; then
        # PG 18: PGDATA is nested under the volume, so mounting <out> as the volume root works -
        # this is what a docker-compose '- ./pgdata:/var/lib/postgresql' expects.
        echo "       docker run -e POSTGRES_PASSWORD=... -v '$OUT_ABS:/var/lib/postgresql' postgres:$RESTORED_MAJOR"
        echo "     (or bind '$DATA_DIR' straight at '$CONTAINER_PGDATA')."
    else
        echo "       docker run -e POSTGRES_PASSWORD=... -v '$DATA_DIR:$CONTAINER_PGDATA' postgres:${RESTORED_MAJOR:-<major>}"
    fi
else
    PG_CTL="pg_ctl"
    [ -n "$PG_BIN" ] && PG_CTL="$PG_BIN/pg_ctl"

    echo "  2. Start PostgreSQL against it (as the postgres user), e.g.:"
    echo "       $PG_CTL -D '$DATA_DIR' start"
    echo "     or point your server's data_directory at it."
fi

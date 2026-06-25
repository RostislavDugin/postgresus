#!/bin/sh
# Databasus physical-restore helper.
#
# Takes a restore bundle produced by the Databasus restore stream, decompresses
# what needs decompressing, runs pg_combinebackup over the full + incremental
# chain, and wires up WAL replay / point-in-time recovery.
#
# Usage:
#   sh databasus-recovery.sh [--pg-bin <dir>] [--target-time <utc-ts>] <bundle-url-or-path> [output-dir]
#
#   --pg-bin <dir>        directory holding the PostgreSQL tools (pg_combinebackup,
#                         pg_ctl) when they are not on PATH.
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
usage: sh databasus-recovery.sh [--pg-bin <dir>] [--target-time <utc-ts>] <bundle-url-or-path> [output-dir]

  --pg-bin <dir>        directory holding the PostgreSQL tools (pg_combinebackup,
                        pg_ctl) when they are not on PATH
  --target-time <ts>    point-in-time recovery target, a PostgreSQL-parseable UTC
                        timestamp (e.g. '2026-06-12 14:30:00+00:00'); omit to
                        replay all shipped WAL. Ignored when the bundle ships no WAL
  <bundle-url-or-path>  an http(s):// restore-stream URL (with its token) or a
                        path to an already-downloaded .tar bundle
  [output-dir]          where to build the restore (default: databasus-restore)
USAGE
}

PG_BIN=""
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
        echo "error: required tool '$1' is not installed or not on PATH" >&2
        exit 1
    fi
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
require_tool pg_combinebackup

mkdir -p "$OUT_DIR"
OUT_ABS="$(cd "$OUT_DIR" && pwd)"
BUNDLE_DIR="$OUT_ABS/bundle"
DATA_DIR="$OUT_ABS/data"
mkdir -p "$BUNDLE_DIR"

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
            require_tool zstd
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

rm -rf "$DATA_DIR"
pg_combinebackup $INPUTS -o "$DATA_DIR"
chmod 700 "$DATA_DIR"

# 5. Wire up WAL replay / PITR when the bundle ships WAL. A per-backup restore
#    ships none - its combined directory is already consistent, so we stop here.
WAL_DIR="$BUNDLE_DIR/wal"
if [ -d "$WAL_DIR" ] && [ -n "$(ls -A "$WAL_DIR" 2>/dev/null)" ]; then
    # WAL segments ship zstd-compressed and are inflated on demand at replay time.
    require_tool zstd
    ZSTD_BIN="$(command -v zstd)"

    # PostgreSQL runs restore_command through `sh -c` with %f/%p substituted, so
    # inline the lookup: decompress a .zst segment on the fly, fall back to a
    # plaintext file (history files), or exit non-zero to signal end-of-archive.
    # Inlining (rather than a helper script in PGDATA) avoids depending on the
    # data directory being on an exec-mounted filesystem. Only double quotes are
    # used inside so the value stays safely wrapped in single quotes.
    restore_command="d=\"$WAL_DIR\"; if [ -f \"\$d/%f.zst\" ]; then \"$ZSTD_BIN\" -dqc \"\$d/%f.zst\" > %p; elif [ -f \"\$d/%f\" ]; then cp \"\$d/%f\" %p; else exit 1; fi"

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
fi

PG_CTL="pg_ctl"
[ -n "$PG_BIN" ] && PG_CTL="$PG_BIN/pg_ctl"

echo
echo "Restore prepared at: $DATA_DIR"
echo "Next steps:"
echo "  1. Ensure the directory is owned by the postgres OS user:"
echo "       chown -R postgres:postgres '$DATA_DIR'"
echo "  2. Start PostgreSQL against it (as the postgres user), e.g.:"
echo "       $PG_CTL -D '$DATA_DIR' start"
echo "     or point your server's data_directory at it."

// Builds the shell commands the restore dialog shows. Pure string assembly - no
// network or React - so the wiring (which flag appears when, host vs Docker) is
// unit-tested in isolation. The served recovery_script.sh accepts optional
// `--pg-bin <dir>` / `--combine-image <image>` flags plus positional
// `<bundle> [output-dir]`; the manual steps mirror what that script does by hand for
// users who would rather not pipe curl into sh. Decompression always runs with the
// host's zstd; on Docker only the version-specific pg_combinebackup runs in a
// container.

export type RestoreEnvironment = 'host' | 'docker';

// PostgreSQL 18 moved the Docker image's PGDATA to a version-specific path and the
// VOLUME up to /var/lib/postgresql; PG <=17 keeps the flat /var/lib/postgresql/data.
// The restored cluster must be bind-mounted at exactly this PGDATA, or the entrypoint
// initializes a fresh empty cluster instead.
export const containerDataDir = (pgVersion: string): string =>
  Number(pgVersion) >= 18 ? `/var/lib/postgresql/${pgVersion}/docker` : '/var/lib/postgresql/data';

export const containerVolumeDir = (pgVersion: string): string =>
  Number(pgVersion) >= 18 ? '/var/lib/postgresql' : '/var/lib/postgresql/data';

// Where the recovery script builds the cluster under the output dir, mirroring the image's
// PGDATA layout so a volume-root mount finds it: PG 18 nests <major>/docker, PG <=17 uses data.
export const clusterDataDir = (outputDir: string, pgVersion: string): string =>
  Number(pgVersion) >= 18 ? `${outputDir}/${pgVersion}/docker` : `${outputDir}/data`;

export interface ScriptCommandParams {
  scriptUrl: string;
  bundleUrl: string;
  outputDir: string;
  pgBin: string;
  // PostgreSQL-parseable UTC timestamp; empty for a "latest" restore.
  targetTime: string;
}

export interface DockerScriptCommandParams {
  scriptUrl: string;
  bundleUrl: string;
  outputDir: string;
  image: string;
  // PostgreSQL-parseable UTC timestamp; empty for a "latest" restore.
  targetTime: string;
}

export interface ManualStepsParams {
  bundleUrl: string;
  outputDir: string;
  // the source cluster's major version - decides the cluster's layout under outputDir.
  pgVersion: string;
  pgBin: string;
  image: string;
  environment: RestoreEnvironment;
  // a per-backup restore ships no WAL; a point-in-time / latest restore does and so
  // needs the recovery-wiring step.
  hasWal: boolean;
  // preformatted timestamp PostgreSQL can parse (empty for a "latest" restore).
  targetTime?: string;
}

export interface RestoreStep {
  title: string;
  code: string;
}

const combineBinary = (pgBin: string): string => {
  const trimmed = pgBin.trim();

  return trimmed ? `"${trimmed}/pg_combinebackup"` : 'pg_combinebackup';
};

// Each backup ships as bundle/<dir>/base.tar<ext> (still compressed) beside its
// backup_manifest. Decompress every one into bundle/recon/<dir> and restore the
// manifest there - exactly what recovery_script.sh does - so pg_combinebackup can
// read decompressed PGDATA directories. Runs on the host (zstd lives there, not in
// the official postgres image).
const decompressBackupsBlock = (): string =>
  [
    `for b in bundle/full $(ls -d bundle/incr-* 2>/dev/null | sort -V); do`,
    `  d="bundle/recon/$(basename "$b")"`,
    `  mkdir -p "$d"`,
    `  blob=$(ls "$b"/base.tar* 2>/dev/null | head -n1)`,
    `  case "$blob" in`,
    `    *.zst) zstd -dqc "$blob" | tar -xf - -C "$d" ;;`,
    `    *.gz)  gzip -dc "$blob" | tar -xf - -C "$d" ;;`,
    `    *)     tar -xf "$blob" -C "$d" ;;`,
    `  esac`,
    `  cp "$b/backup_manifest" "$d/backup_manifest"`,
    `done`,
  ].join('\n');

// `bundle/recon/full` plus every `bundle/recon/incr-N` in numeric order - the inputs
// pg_combinebackup needs, oldest to newest. Output goes to the version-specific cluster dir.
const combineCommand = (pgBin: string, dataDir: string): string =>
  `${combineBinary(pgBin)} bundle/recon/full $(ls -d bundle/recon/incr-* 2>/dev/null | sort -V) -o "${dataDir}"`;

// Reads the restored cluster's control file, where the source's recovery-required parameters live.
// Runs pg_controldata the same way the reconstruct step runs pg_combinebackup - in the container on
// Docker, off the host tools otherwise.
const controlDataCommand = (
  environment: RestoreEnvironment,
  image: string,
  pgBin: string,
  dataDir: string,
): string => {
  if (environment === 'docker') {
    return `docker run --rm -v "$PWD:/work" -w /work ${image} pg_controldata "${dataDir}"`;
  }

  const trimmed = pgBin.trim();
  const binary = trimmed ? `"${trimmed}/pg_controldata"` : 'pg_controldata';

  return `${binary} "${dataDir}"`;
};

export const buildScriptCommand = ({
  scriptUrl,
  bundleUrl,
  outputDir,
  pgBin,
  targetTime,
}: ScriptCommandParams): string => {
  const pgBinArg = pgBin.trim() ? `--pg-bin "${pgBin.trim()}" ` : '';
  const targetArg = targetTime.trim() ? `--target-time "${targetTime.trim()}" ` : '';

  return `curl -fsSL "${scriptUrl}" | sh -s -- ${pgBinArg}${targetArg}"${bundleUrl}" "${outputDir}"`;
};

// Runs the script on the host (host zstd decompresses the bundle); only
// pg_combinebackup runs in the container via --combine-image. The script downloads
// the bundle itself, so the single-use token is consumed exactly once.
export const buildDockerScriptCommand = ({
  scriptUrl,
  bundleUrl,
  outputDir,
  image,
  targetTime,
}: DockerScriptCommandParams): string => {
  const targetArg = targetTime.trim() ? `--target-time "${targetTime.trim()}" ` : '';

  return `curl -fsSL "${scriptUrl}" | sh -s -- --combine-image "${image}" ${targetArg}"${bundleUrl}" "${outputDir}"`;
};

export const buildManualSteps = ({
  bundleUrl,
  outputDir,
  pgVersion,
  pgBin,
  image,
  environment,
  hasWal,
  targetTime,
}: ManualStepsParams): RestoreStep[] => {
  const dataDir = clusterDataDir(outputDir, pgVersion);
  const combine =
    environment === 'docker'
      ? [
          decompressBackupsBlock(),
          // --user keeps the output host-owned so the chmod and WAL-wiring below run
          // on the host without sudo; only pg_combinebackup needs the container.
          `docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" -w /work ${image} \\`,
          `  ${combineCommand('', dataDir)}`,
          `chmod 700 "${dataDir}"`,
        ].join('\n')
      : [decompressBackupsBlock(), combineCommand(pgBin, dataDir), `chmod 700 "${dataDir}"`].join(
          '\n',
        );

  const steps: RestoreStep[] = [
    {
      title: 'Download the bundle',
      code: `curl -fsSL "${bundleUrl}" -o restore.tar`,
    },
    {
      title: 'Extract it',
      code: `mkdir -p bundle\ntar -xf restore.tar -C bundle`,
    },
    {
      title: 'Verify the transfer',
      code: `(cd bundle && sha256sum -c MANIFEST.sha256)`,
    },
    {
      title: 'Reconstruct the data directory',
      code: combine,
    },
  ];

  if (hasWal) {
    const targetLines = targetTime
      ? `\n  echo "recovery_target_time = '${targetTime}'"\n  echo "recovery_target_action = 'promote'"`
      : '';

    steps.push({
      title: 'Decompress WAL and wire up recovery',
      // WAL is inflated inside PGDATA and restore_command is relative to it, so replay works
      // whether the cluster starts on the host or in a container (the WAL travels with PGDATA).
      code: [
        `mkdir -p "${dataDir}/databasus_wal_restore"`,
        `for f in bundle/wal/*; do`,
        `  [ -e "$f" ] || continue`,
        `  case "$f" in`,
        `    *.zst) zstd -dq "$f" -o "${dataDir}/databasus_wal_restore/$(basename "\${f%.zst}")" ;;`,
        `    *) cp "$f" "${dataDir}/databasus_wal_restore/" ;;`,
        `  esac`,
        `done`,
        `{`,
        `  echo "restore_command = 'cp \\"databasus_wal_restore/%f\\" \\"%p\\"'"${targetLines}`,
        `} >> "${dataDir}/postgresql.auto.conf"`,
        `touch "${dataDir}/recovery.signal"`,
      ].join('\n'),
    });

    steps.push({
      title: 'If recovery stops on parameter settings',
      // Archive recovery aborts when these are below the source's values - the served script sets
      // them from the control file automatically, but a hand-run restore must do it too.
      code: [
        `# Read the source's recovery-required parameters from the restored control file:`,
        controlDataCommand(environment, image, pgBin, dataDir),
        `# then set each at or above the printed value in "${dataDir}/postgresql.auto.conf" and restart:`,
        `#   max_connections, max_worker_processes, max_wal_senders,`,
        `#   max_prepared_transactions, max_locks_per_transaction`,
      ].join('\n'),
    });
  }

  return steps;
};

import { describe, expect, it } from 'vitest';

import {
  buildDockerScriptCommand,
  buildManualSteps,
  buildScriptCommand,
  clusterDataDir,
  containerDataDir,
  containerVolumeDir,
} from './restoreCommands';

const scriptUrl = 'https://app.example.com/api/v1/backups/physical/recovery-script';
const bundleUrl = 'https://app.example.com/api/v1/backups/physical/restore-stream?token=abc';

describe('container paths', () => {
  it('uses the flat data path for PostgreSQL 17 and below', () => {
    expect(containerDataDir('17')).toBe('/var/lib/postgresql/data');
    expect(containerVolumeDir('17')).toBe('/var/lib/postgresql/data');
  });

  it('uses the version-specific PGDATA and volume root for PostgreSQL 18', () => {
    expect(containerDataDir('18')).toBe('/var/lib/postgresql/18/docker');
    expect(containerVolumeDir('18')).toBe('/var/lib/postgresql');
  });

  it('builds the cluster under the output dir mirroring the image layout', () => {
    expect(clusterDataDir('./out', '17')).toBe('./out/data');
    expect(clusterDataDir('./out', '18')).toBe('./out/18/docker');
  });
});

describe('buildScriptCommand', () => {
  it('omits --pg-bin and --target-time when neither is set', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      pgBin: '',
      targetTime: '',
    });

    expect(command).not.toContain('--pg-bin');
    expect(command).not.toContain('--target-time');
    expect(command).toContain(`"${bundleUrl}" "./databasus-restore"`);
  });

  it('includes --pg-bin when a bin path is set', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      pgBin: '/usr/lib/postgresql/18/bin',
      targetTime: '',
    });

    expect(command).toContain('--pg-bin "/usr/lib/postgresql/18/bin"');
  });

  it('includes --target-time when a target time is set', () => {
    const command = buildScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      pgBin: '',
      targetTime: '2026-06-12 14:30:00+00:00',
    });

    expect(command).toContain('--target-time "2026-06-12 14:30:00+00:00"');
  });
});

describe('buildDockerScriptCommand', () => {
  it('runs the script on the host, combines via the chosen image, and references the bundle once', () => {
    const command = buildDockerScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      image: 'postgres:18',
      targetTime: '',
    });

    expect(command).toContain(`curl -fsSL "${scriptUrl}" | sh -s --`);
    expect(command).toContain('--combine-image "postgres:18"');
    expect(command).not.toContain('--target-time');
    expect(command.match(/restore-stream/g)).toHaveLength(1);
  });

  it('passes --target-time to the host script when a target time is set', () => {
    const command = buildDockerScriptCommand({
      scriptUrl,
      bundleUrl,
      outputDir: './databasus-restore',
      image: 'postgres:18',
      targetTime: '2026-06-12 14:30:00+00:00',
    });

    expect(command).toContain(
      '--combine-image "postgres:18" --target-time "2026-06-12 14:30:00+00:00"',
    );
  });
});

describe('buildManualSteps', () => {
  const base = {
    bundleUrl,
    outputDir: './databasus-restore',
    pgVersion: '18',
    pgBin: '',
    image: 'postgres:18',
    targetTime: '',
  };

  it('decompresses each backup blob before pg_combinebackup and skips recovery when there is no WAL', () => {
    const steps = buildManualSteps({ ...base, environment: 'host', hasWal: false });
    const titles = steps.map((step) => step.title);
    const combine = steps.find((step) => step.title === 'Reconstruct the data directory');

    expect(combine?.code).toContain('base.tar');
    expect(combine?.code).toContain('pg_combinebackup bundle/recon/full');
    expect(combine!.code.indexOf('base.tar')).toBeLessThan(
      combine!.code.indexOf('pg_combinebackup'),
    );
    expect(titles).not.toContain('Decompress WAL and wire up recovery');
    expect(titles).not.toContain('If recovery stops on parameter settings');
  });

  it('adds a recovery-parameter note pointing at pg_controldata when there is WAL', () => {
    const host = buildManualSteps({ ...base, environment: 'host', hasWal: true });
    const docker = buildManualSteps({ ...base, environment: 'docker', hasWal: true });
    const hostNote = host.find((step) => step.title === 'If recovery stops on parameter settings');
    const dockerNote = docker.find(
      (step) => step.title === 'If recovery stops on parameter settings',
    );

    expect(hostNote?.code).toContain('pg_controldata "./databasus-restore/18/docker"');
    expect(hostNote?.code).toContain('max_connections');
    expect(dockerNote?.code).toContain(
      'docker run --rm -v "$PWD:/work" -w /work postgres:18 pg_controldata',
    );
  });

  it('builds the cluster at the version-specific path', () => {
    const pg18 = buildManualSteps({ ...base, pgVersion: '18', environment: 'host', hasWal: false });
    const pg17 = buildManualSteps({ ...base, pgVersion: '17', environment: 'host', hasWal: false });
    const combine18 = pg18.find((step) => step.title === 'Reconstruct the data directory');
    const combine17 = pg17.find((step) => step.title === 'Reconstruct the data directory');

    expect(combine18?.code).toContain('-o "./databasus-restore/18/docker"');
    expect(combine17?.code).toContain('-o "./databasus-restore/data"');
  });

  it('includes the WAL recovery step with a PGDATA-relative restore_command when there is WAL', () => {
    const steps = buildManualSteps({ ...base, environment: 'host', hasWal: true });
    const recovery = steps.find((step) => step.title === 'Decompress WAL and wire up recovery');

    expect(recovery).toBeDefined();
    expect(recovery?.code).toContain('./databasus-restore/18/docker/recovery.signal');
    expect(recovery?.code).toContain('databasus_wal_restore/%f');
    // restore_command is relative to PGDATA (no host-absolute path) so replay is container-portable.
    expect(recovery?.code).not.toContain('wal_abs');
  });

  it('bakes the target time into recovery settings only when one is given', () => {
    const withTarget = buildManualSteps({
      ...base,
      environment: 'host',
      hasWal: true,
      targetTime: '2026-06-12 14:30:00+00:00',
    });
    const recovery = withTarget.find(
      (step) => step.title === 'Decompress WAL and wire up recovery',
    );

    expect(recovery?.code).toContain("recovery_target_time = '2026-06-12 14:30:00+00:00'");
    expect(recovery?.code).toContain("recovery_target_action = 'promote'");

    const withoutTarget = buildManualSteps({ ...base, environment: 'host', hasWal: true });
    const latestRecovery = withoutTarget.find(
      (step) => step.title === 'Decompress WAL and wire up recovery',
    );

    expect(latestRecovery?.code).not.toContain('recovery_target_time');
  });

  it('decompresses on the host then runs pg_combinebackup through docker in the docker environment', () => {
    const steps = buildManualSteps({ ...base, environment: 'docker', hasWal: false });
    const combine = steps.find((step) => step.title === 'Reconstruct the data directory');

    expect(combine?.code).toContain(
      'docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" -w /work postgres:18',
    );
    expect(combine?.code).toContain('pg_combinebackup bundle/recon/full');
    expect(combine!.code.indexOf('base.tar')).toBeLessThan(combine!.code.indexOf('docker run'));
  });
});

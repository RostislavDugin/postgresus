import { CheckOutlined, CopyOutlined } from '@ant-design/icons';
import { Alert, Button, DatePicker, Input, Segmented, Spin, Tabs } from 'antd';
import type { Dayjs } from 'dayjs';
import { type JSX, useEffect, useState } from 'react';

import { getApplicationServer } from '../../../../constants';
import {
  type PhysicalBackupListItem,
  physicalBackupsApi,
} from '../../../../entity/backups/physical';
import type { Database } from '../../../../entity/databases';
import { ClipboardHelper } from '../../../../shared/lib/ClipboardHelper';
import {
  type RestoreEnvironment,
  buildDockerScriptCommand,
  buildManualSteps,
  buildScriptCommand,
  clusterDataDir,
  containerDataDir,
  containerVolumeDir,
} from '../lib/restoreCommands';

interface Props {
  database: Database;
  backup?: PhysicalBackupListItem;
  onClose: () => void;
}

type RestoreMethod = 'script' | 'manual';

// Surfaces a friendlier hint for the two known API failures: a concurrent download
// (409) and an unreachable target time / WAL gap (422).
const describeRestoreError = (message: string): string => {
  if (message.includes('409') || message.toLowerCase().includes('in progress')) {
    return `${message}\n\nA restore download is already in progress for this database. Wait for it to finish, then try again.`;
  }

  if (message.includes('422') || message.toLowerCase().includes('gap')) {
    return `${message}\n\nThe requested target time cannot be reached - there is a WAL gap or the time is out of the available range. Pick a different time or restore the latest available point.`;
  }

  return message;
};

interface CopyableCommandProps {
  id: string;
  title: string;
  code: string;
  copiedKey: string | null;
  onCopy: (id: string, code: string) => void;
}

const CopyableCommand = ({
  id,
  title,
  code,
  copiedKey,
  onCopy,
}: CopyableCommandProps): JSX.Element => {
  const isCopied = copiedKey === id;

  return (
    <div className="mt-3">
      <div className="mb-1 flex items-center justify-between">
        <div className="text-xs font-medium text-gray-600 dark:text-gray-300">{title}</div>
        <Button
          size="small"
          type="text"
          icon={isCopied ? <CheckOutlined /> : <CopyOutlined />}
          onClick={() => onCopy(id, code)}
        >
          {isCopied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded bg-gray-100 p-3 text-xs whitespace-pre-wrap text-gray-700 dark:bg-gray-700 dark:text-gray-200">
        {code}
      </pre>
    </div>
  );
};

export const PhysicalRestoreComponent = ({ database, backup, onClose }: Props): JSX.Element => {
  // The source cluster's major version, detected on connect. Physical databases are
  // always PostgreSQL 17 or 18; the fallback is only a defensive guard. Every shown
  // image tag, bin path and container PGDATA derives from this single value.
  const pgVersion = database.postgresqlPhysical?.version ?? '17';

  const [targetTime, setTargetTime] = useState<Dayjs | undefined>();
  const [isGenerating, setIsGenerating] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string>();
  const [bundleUrl, setBundleUrl] = useState<string>();
  const [restoreMethod, setRestoreMethod] = useState<RestoreMethod>('script');
  const [environment, setEnvironment] = useState<RestoreEnvironment>('host');
  const [outputDir, setOutputDir] = useState('./databasus-restore');
  const [pgBin, setPgBin] = useState('');
  const [dockerImage, setDockerImage] = useState(`postgres:${pgVersion}`);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const generateRestore = async () => {
    setIsGenerating(true);
    setErrorMessage(undefined);
    setBundleUrl(undefined);
    setCopiedKey(null);

    try {
      const response = backup
        ? await physicalBackupsApi.generateBackupRestoreToken(backup.id)
        : await physicalBackupsApi.generatePitrRestoreToken(
            database.id,
            targetTime ? targetTime.utc().toISOString() : undefined,
          );

      setBundleUrl(`${getApplicationServer()}${response.url}`);
    } catch (e) {
      setErrorMessage(describeRestoreError((e as Error).message));
    }

    setIsGenerating(false);
  };

  const copyText = async (id: string, text: string) => {
    await ClipboardHelper.copyToClipboard(text);
    setCopiedKey(id);
    setTimeout(() => setCopiedKey((current) => (current === id ? null : current)), 2000);
  };

  // A token is minted automatically: once on open for a per-backup restore, and
  // again whenever the PITR target changes. The token is single-use and the stream
  // is unauthenticated, so it must be a fresh capability the user can curl - not a
  // static command.
  useEffect(() => {
    generateRestore();
  }, [targetTime]);

  const hasWal = backup === undefined;
  const dataDir = clusterDataDir(outputDir, pgVersion);
  const scriptUrl = `${getApplicationServer()}/api/v1/backups/physical/recovery-script`;
  const recoveryTargetTime = targetTime ? targetTime.utc().format('YYYY-MM-DD HH:mm:ssZ') : '';

  const renderConfig = (env: RestoreEnvironment): JSX.Element => (
    <div className="mb-3">
      <div className="mb-2 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Restore directory</div>
        <Input
          value={outputDir}
          onChange={(e) => setOutputDir(e.target.value)}
          className="w-full max-w-[320px]"
        />
      </div>
      {env === 'host' ? (
        <div className="flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">PostgreSQL bin path</div>
          <Input
            value={pgBin}
            onChange={(e) => setPgBin(e.target.value)}
            placeholder={`/usr/lib/postgresql/${pgVersion}/bin (optional)`}
            className="w-full max-w-[320px]"
          />
        </div>
      ) : (
        <div className="flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">PostgreSQL image</div>
          <Input
            value={dockerImage}
            onChange={(e) => setDockerImage(e.target.value)}
            className="w-full max-w-[320px]"
          />
        </div>
      )}
    </div>
  );

  const renderCommands = (env: RestoreEnvironment, url: string): JSX.Element => {
    if (restoreMethod === 'script') {
      const code =
        env === 'host'
          ? buildScriptCommand({
              scriptUrl,
              bundleUrl: url,
              outputDir,
              pgBin,
              targetTime: recoveryTargetTime,
            })
          : buildDockerScriptCommand({
              scriptUrl,
              bundleUrl: url,
              outputDir,
              image: dockerImage,
              targetTime: recoveryTargetTime,
            });

      return (
        <CopyableCommand
          id={`script-${env}`}
          title={env === 'host' ? 'Run on your restore host' : 'Run where Docker is available'}
          code={code}
          copiedKey={copiedKey}
          onCopy={copyText}
        />
      );
    }

    const steps = buildManualSteps({
      bundleUrl: url,
      outputDir,
      pgVersion,
      pgBin,
      image: dockerImage,
      environment: env,
      hasWal,
      targetTime: recoveryTargetTime,
    });

    return (
      <>
        {steps.map((step, index) => (
          <CopyableCommand
            key={step.title}
            id={`manual-${env}-${index}`}
            title={`${index + 1}. ${step.title}`}
            code={step.code}
            copiedKey={copiedKey}
            onCopy={copyText}
          />
        ))}
      </>
    );
  };

  const renderEnvironmentPanel = (env: RestoreEnvironment, url: string): JSX.Element => (
    <div>
      {renderConfig(env)}
      <Alert
        type="info"
        showIcon
        message="Before you run"
        description={
          env === 'host' ? (
            <ul className="ml-4 list-disc">
              <li>
                Install the PostgreSQL {pgVersion} client tools - set the bin path above if they are
                not on PATH.
              </li>
              <li>zstd is required to decompress the backup{hasWal ? ' and its WAL' : ''}.</li>
              <li>The restore directory must be empty and not an in-use cluster.</li>
            </ul>
          ) : (
            <ul className="ml-4 list-disc">
              <li>Use a postgres:{pgVersion} image - the major version must match the source.</li>
              <li>
                The host needs zstd, tar and curl: the download and all decompression - the base
                backup{hasWal ? ' and the WAL' : ''} - run on the host, and only pg_combinebackup
                runs in the container, so the postgres image needs no extra tools.
              </li>
            </ul>
          )
        }
      />
      {renderCommands(env, url)}
      <div className="mt-3" />
      <Alert
        type="info"
        showIcon
        message="After it finishes"
        description={
          env === 'host' ? (
            <ul className="ml-4 list-disc">
              <li>
                The cluster is at <code>{dataDir}</code>.
              </li>
              <li>
                Own it as root: <code>chown -R postgres:postgres {dataDir}</code>.
              </li>
              <li>
                Start it as postgres: <code>pg_ctl -D {dataDir} start</code>, or point
                data_directory at it.
              </li>
              {hasWal && (
                <li>
                  PostgreSQL replays WAL and promotes - watch the log for recovery completion.
                </li>
              )}
            </ul>
          ) : (
            <ul className="ml-4 list-disc">
              <li>
                <code>{dataDir}</code> on the host is the restored cluster.
              </li>
              {Number(pgVersion) >= 18 ? (
                <li>
                  PostgreSQL {pgVersion} keeps its data directory at{' '}
                  <code>{containerDataDir(pgVersion)}</code>, under the volume{' '}
                  <code>{containerVolumeDir(pgVersion)}</code>. Mount the output dir as that volume
                  root - this is what a docker-compose <code>./pgdata:/var/lib/postgresql</code>{' '}
                  expects:
                  <br />
                  <code>
                    {`docker run -e POSTGRES_PASSWORD=... -v "$PWD/${outputDir}:${containerVolumeDir(pgVersion)}" postgres:${pgVersion}`}
                  </code>
                </li>
              ) : (
                <li>
                  Bind-mount the cluster at the image&apos;s data directory{' '}
                  <code>{containerDataDir(pgVersion)}</code>:
                  <br />
                  <code>
                    {`docker run -e POSTGRES_PASSWORD=... -v "$PWD/${dataDir}:${containerDataDir(pgVersion)}" postgres:${pgVersion}`}
                  </code>
                </li>
              )}
              <li>Ownership must match the postgres uid inside the image.</li>
            </ul>
          )
        }
      />
    </div>
  );

  return (
    <div>
      {backup ? (
        <div className="mb-3 text-sm text-gray-600 dark:text-gray-400">
          Restore command for this backup. A full backup restores itself; an incremental restores
          its full backup and all incremental ancestors.
        </div>
      ) : (
        <>
          <div className="mb-3 text-sm text-gray-600 dark:text-gray-400">
            Point-in-time restore. Pick a target time, or leave it empty to restore the latest
            available point.
          </div>
          <div className="mb-3 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[120px] sm:mb-0">Target time</div>
            <DatePicker
              showTime
              value={targetTime}
              onChange={(value) => setTargetTime(value ?? undefined)}
              className="w-full max-w-[260px] grow"
              placeholder="Latest available"
            />
          </div>
        </>
      )}

      {errorMessage && (
        <div className="mt-3 rounded border border-red-300/50 bg-red-50 px-3 py-2 text-sm whitespace-pre-line text-red-700 dark:border-red-600/30 dark:bg-red-900/20 dark:text-red-400">
          {errorMessage}
        </div>
      )}

      {isGenerating && (
        <div className="mt-5 flex items-center gap-2 border-t border-gray-200 pt-4 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400">
          <Spin size="small" />
          Preparing restore command...
        </div>
      )}

      {!isGenerating && bundleUrl && (
        <div className="mt-5 border-t border-gray-200 pt-4 dark:border-gray-700">
          <Segmented<RestoreMethod>
            value={restoreMethod}
            onChange={setRestoreMethod}
            options={[
              { label: 'Via script', value: 'script' },
              { label: 'Manual', value: 'manual' },
            ]}
          />
          <Tabs
            className="mt-2"
            activeKey={environment}
            onChange={(key) => setEnvironment(key as RestoreEnvironment)}
            items={[
              {
                key: 'host',
                label: 'Host PostgreSQL',
                children: renderEnvironmentPanel('host', bundleUrl),
              },
              {
                key: 'docker',
                label: 'Docker',
                children: renderEnvironmentPanel('docker', bundleUrl),
              },
            ]}
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            The download link is single-use and expires in 15 minutes. If a command fails with a
            401, reopen this dialog to mint a fresh one.
          </p>
        </div>
      )}

      <div className="mt-4 flex">
        <Button className="ml-auto" onClick={onClose}>
          Close
        </Button>
      </div>
    </div>
  );
};

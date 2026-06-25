import { ConnectionErrorCode } from './ConnectionErrorCode';

export interface PhysicalConnectionErrorTextRun {
  text: string;
  isBold?: boolean;
}

export type PhysicalConnectionErrorStep =
  | { type: 'command'; command: string }
  | { type: 'note'; runs: PhysicalConnectionErrorTextRun[] };

export interface PhysicalConnectionErrorContent {
  title: string;
  summary: string;
  buildSteps?: (ctx: { username: string }) => PhysicalConnectionErrorStep[];
  // Note about managed PostgreSQL (RDS / Azure / GCP). Rendered below the steps, since on managed
  // providers the commands do not apply and the operator uses the provider console instead.
  managedNote?: string | ((ctx: { username: string }) => string);
}

// pg_hba.conf accepts the literal "all" in the address field to match every host. We use it because
// the exact IP Databasus connects from lives only in the server-side rejection, which the backend no
// longer forwards - the operator can narrow it to a CIDR range afterwards if they want.
const REPLICATION_HBA_ADDRESS = 'all';

export const physicalConnectionErrorContent: Record<
  ConnectionErrorCode,
  PhysicalConnectionErrorContent
> = {
  [ConnectionErrorCode.PgHbaNoEntry]: {
    title: 'Replication is not allowed from this host',
    summary:
      'PostgreSQL has no pg_hba.conf entry that permits a replication connection from Databasus. Physical backups stream via replication, which needs a "host replication" rule - an ordinary "host all" rule does not cover it.',
    buildSteps: () => [
      {
        type: 'note',
        runs: [
          {
            text: 'Add the line below to pg_hba.conf. The first "all" matches any user and the second any host - narrow the user to the specific replication role and/or the host to a CIDR range if you want to restrict it.',
          },
        ],
      },
      {
        type: 'command',
        command: `host    replication    all    ${REPLICATION_HBA_ADDRESS}    scram-sha-256`,
      },
      {
        type: 'note',
        runs: [
          { text: 'After updating pg_hba.conf, ' },
          { text: 'reload PG', isBold: true },
          { text: '.' },
        ],
      },
    ],
    managedNote:
      'On managed PostgreSQL (RDS / Azure / GCP) you cannot edit pg_hba.conf - enable external or replication access in the provider console instead.',
  },
  [ConnectionErrorCode.BadCredentials]: {
    title: 'Wrong username or password',
    summary:
      'PostgreSQL rejected the credentials. Double-check the username and password for this database.',
  },
  [ConnectionErrorCode.NoReplicationPrivilege]: {
    title: 'User cannot run replication',
    summary:
      'The user connected but lacks the REPLICATION privilege that physical backups require.',
    buildSteps: ({ username }) => [
      { type: 'note', runs: [{ text: 'Grant it with the command below (run as a superuser).' }] },
      { type: 'command', command: `ALTER ROLE ${username} REPLICATION;` },
    ],
    managedNote: ({ username }) =>
      `On AWS RDS use GRANT rds_replication TO ${username}; on Azure / GCP enable replication for the role in the provider console.`,
  },
  [ConnectionErrorCode.WalLevelInvalid]: {
    title: 'wal_level is too low',
    summary:
      'wal_level must be "replica" or "logical" for physical backups. It is currently set lower.',
    buildSteps: () => [
      {
        type: 'note',
        runs: [
          {
            text: 'Apply the change below, then restart PostgreSQL (wal_level only takes effect after a restart).',
          },
        ],
      },
      { type: 'command', command: 'ALTER SYSTEM SET wal_level = replica;' },
    ],
    managedNote: 'On managed PostgreSQL set wal_level in the provider parameter group.',
  },
  [ConnectionErrorCode.NoWalSenders]: {
    title: 'No WAL sender processes available',
    summary: 'max_wal_senders is 0, so PostgreSQL cannot stream WAL for backups.',
    buildSteps: () => [
      { type: 'note', runs: [{ text: 'Apply the change below, then restart PostgreSQL.' }] },
      { type: 'command', command: 'ALTER SYSTEM SET max_wal_senders = 10;' },
    ],
    managedNote: 'On managed PostgreSQL set max_wal_senders in the provider parameter group.',
  },
  [ConnectionErrorCode.NoReplicationSlots]: {
    title: 'No replication slots available',
    summary: 'max_replication_slots is 0, so PostgreSQL cannot allocate a slot for backups.',
    buildSteps: () => [
      { type: 'note', runs: [{ text: 'Apply the change below, then restart PostgreSQL.' }] },
      { type: 'command', command: 'ALTER SYSTEM SET max_replication_slots = 10;' },
    ],
    managedNote: 'On managed PostgreSQL set max_replication_slots in the provider parameter group.',
  },
  [ConnectionErrorCode.WalSummaryDisabled]: {
    title: 'WAL summarization is off',
    summary:
      'summarize_wal must be on for incremental backups. It is currently off on this server.',
    buildSteps: () => [
      { type: 'note', runs: [{ text: 'Apply the change below (no restart needed).' }] },
      { type: 'command', command: 'ALTER SYSTEM SET summarize_wal = on;' },
      { type: 'note', runs: [{ text: 'and then' }] },
      { type: 'command', command: 'SELECT pg_reload_conf();' },
    ],
    managedNote: 'On managed PostgreSQL set summarize_wal in the provider parameter group.',
  },
  [ConnectionErrorCode.CustomTablespaces]: {
    title: 'Custom tablespaces are not supported',
    summary:
      'This cluster has tablespaces outside pg_default / pg_global, which physical backups cannot stream. Drop the custom tablespaces, or switch this database to logical backups.',
  },
  [ConnectionErrorCode.SystemIdentifierMismatch]: {
    title: 'This is a different cluster',
    summary:
      'The cluster at this address has a different system identifier than the one this backup was set up for. Point Databasus at the original cluster, or create a new database entry.',
  },
  [ConnectionErrorCode.ConnectionFailed]: {
    title: 'Could not connect',
    summary:
      'Databasus could not open a replication connection. Check the host, port, SSL mode and that a firewall is not blocking the connection.',
  },
};

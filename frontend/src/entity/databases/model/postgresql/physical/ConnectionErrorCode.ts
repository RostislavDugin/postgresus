// Mirrors the backend ConnectionErrorCode (postgresql_shared). The backend returns only this
// code on a failed physical replication test; all user-facing copy lives in the frontend.
export enum ConnectionErrorCode {
  ConnectionFailed = 'connection_failed',
  PgHbaNoEntry = 'pg_hba_no_entry',
  BadCredentials = 'bad_credentials',
  NoReplicationPrivilege = 'no_replication_privilege',
  WalLevelInvalid = 'wal_level_invalid',
  NoWalSenders = 'no_wal_senders',
  NoReplicationSlots = 'no_replication_slots',
  WalSummaryDisabled = 'wal_summary_disabled',
  CustomTablespaces = 'custom_tablespaces',
  SystemIdentifierMismatch = 'system_identifier_mismatch',
}

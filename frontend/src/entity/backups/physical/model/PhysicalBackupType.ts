// Discriminator for a row in the flat physical-backup list. WAL exists only as a
// list/UI tag for a committed WAL segment - it is never a backup the scheduler
// creates, and the physical UI does not render WAL rows.
export enum PhysicalBackupType {
  FULL = 'FULL',
  INCREMENTAL = 'INCREMENTAL',
  WAL = 'WAL',
}

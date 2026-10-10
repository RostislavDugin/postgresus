export interface ScheduledBackupTime {
  scheduledAt: Date;
  // set by the server when the scheduler already treats the backup as due, so a
  // client clock running behind the server cannot postpone it
  isDue: boolean;
}

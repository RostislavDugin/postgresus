export interface TableHealthEntry {
  name: string;
  engine: string;
  status: 'OK' | 'REPAIRED' | 'REPAIR_FAILED' | 'MISSING_FROM_DUMP' | 'CHECK_ERROR';
  message?: string;
  wasRepaired?: boolean;
}

export interface TableHealthReport {
  totalTables: number;
  dumpedTables: number;
  missingTables?: string[];
  repairedCount: number;
  failedRepairs: number;
  tables?: TableHealthEntry[];
  checkedAt: string;
}

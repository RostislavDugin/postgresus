import type { AuditLog } from './AuditLog';

export interface GetAuditLogsResponse {
  auditLogs: AuditLog[];
  total: number;
  limit: number;
  offset: number;
}

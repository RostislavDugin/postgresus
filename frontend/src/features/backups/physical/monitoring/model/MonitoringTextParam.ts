import type { LocalizedText } from '../../../../../shared/i18n';

export type MonitoringTextParam =
  | { kind: 'bytes'; bytes: number }
  | { kind: 'bytesPerSecond'; bytesPerSecond: number }
  | { kind: 'count'; count: number }
  | { kind: 'text'; text: LocalizedText };

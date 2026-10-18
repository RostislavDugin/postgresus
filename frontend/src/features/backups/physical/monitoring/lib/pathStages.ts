import {
  WalStreamHealthLevel,
  type WalStreamReason,
  WalStreamReasonLevel,
} from '../../../../../entity/backups/physical';
import type { AmountProgress } from '../model/AmountProgress';
import { MonitoringTone } from '../model/MonitoringTone';
import { PathStage } from '../model/PathStage';
import { REASON_STAGES } from './reasonTexts';

// Each stage takes the color of the most severe reason about it. Notices never color a stage, and
// without a verdict about streaming there is nothing to color.
export const getStageTones = (
  level: WalStreamHealthLevel,
  reasons: WalStreamReason[],
): Record<PathStage, MonitoringTone> => {
  const isVerdictAboutStreaming =
    level !== WalStreamHealthLevel.NOT_STREAMING && level !== WalStreamHealthLevel.COLLECTING_DATA;
  const defaultTone = isVerdictAboutStreaming ? MonitoringTone.SUCCESS : MonitoringTone.NEUTRAL;

  const stageTones: Record<PathStage, MonitoringTone> = {
    [PathStage.SOURCE]: defaultTone,
    [PathStage.DATABASUS]: defaultTone,
    [PathStage.STORAGE]: defaultTone,
  };

  if (!isVerdictAboutStreaming) {
    return stageTones;
  }

  for (const reason of reasons) {
    const stage = REASON_STAGES[reason.code];

    if (reason.level === WalStreamReasonLevel.BROKEN) {
      stageTones[stage] = MonitoringTone.ERROR;
    } else if (
      reason.level === WalStreamReasonLevel.NEEDS_ATTENTION &&
      stageTones[stage] !== MonitoringTone.ERROR
    ) {
      stageTones[stage] = MonitoringTone.WARNING;
    }
  }

  return stageTones;
};

// An amount above its limit is shown as over the limit rather than as a bar that merely looks
// full, so the reader sees that the limit was crossed.
export const getAmountProgress = (valueBytes: number, limitBytes: number): AmountProgress => {
  if (limitBytes <= 0) {
    return { percent: 0, isOverLimit: false };
  }

  if (valueBytes > limitBytes) {
    return { percent: 100, isOverLimit: true };
  }

  return { percent: Math.max(0, (valueBytes / limitBytes) * 100), isOverLimit: false };
};

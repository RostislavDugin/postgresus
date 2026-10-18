import {
  WAL_STREAM_NOTICE_CODE_LABEL_KEYS,
  WAL_STREAM_REASON_CODE_LABEL_KEYS,
  type WalStreamNotice,
  WalStreamNoticeCode,
  type WalStreamReason,
  WalStreamReasonCode,
  WalStreamReasonLevel,
} from '../../../../../entity/backups/physical';
import type { TranslationKey } from '../../../../../shared/i18n';
import type { MonitoringText } from '../model/MonitoringText';
import type { MonitoringTextParam } from '../model/MonitoringTextParam';
import { PathStage } from '../model/PathStage';
import type { ReasonTexts } from '../model/ReasonTexts';
import { getRoundedDuration, getSecondsBetween } from './roundedDuration';

export const REASON_STAGES: Record<WalStreamReasonCode, PathStage> = {
  [WalStreamReasonCode.STREAMING_STOPPED]: PathStage.DATABASUS,
  [WalStreamReasonCode.SOURCE_UNREACHABLE]: PathStage.SOURCE,
  [WalStreamReasonCode.WAL_CHAIN_BROKEN]: PathStage.STORAGE,
  [WalStreamReasonCode.SLOT_MISSING]: PathStage.SOURCE,
  [WalStreamReasonCode.SLOT_LOST]: PathStage.SOURCE,
  [WalStreamReasonCode.ARCHIVE_NOT_ADVANCING]: PathStage.STORAGE,
  [WalStreamReasonCode.ARCHIVE_SLOWER_THAN_SOURCE]: PathStage.STORAGE,
  [WalStreamReasonCode.SLOT_KEEPS_EXTRA_WAL]: PathStage.SOURCE,
  [WalStreamReasonCode.SLOT_LIMIT_APPROACHING]: PathStage.SOURCE,
  [WalStreamReasonCode.NEAR_SLOT_REBUILD]: PathStage.SOURCE,
  [WalStreamReasonCode.LOCAL_QUEUE_FILLING]: PathStage.DATABASUS,
  [WalStreamReasonCode.FREQUENT_RESTARTS]: PathStage.DATABASUS,
  [WalStreamReasonCode.FORCED_ROTATION_REFUSED]: PathStage.SOURCE,
};

interface ReasonSentenceKeys {
  whatHappened: TranslationKey;
  whatToDo: TranslationKey;

  // Asked instead of whatToDo while a full backup is requested or running, for the conditions
  // only a new full backup repairs.
  whatToDoWhilePending?: TranslationKey;
}

const REASON_SENTENCE_KEYS: Record<WalStreamReasonCode, ReasonSentenceKeys> = {
  [WalStreamReasonCode.STREAMING_STOPPED]: {
    whatHappened: 'backups.physical.monitoring.reasons.streamingStopped.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.streamingStopped.whatToDo',
  },
  [WalStreamReasonCode.SOURCE_UNREACHABLE]: {
    whatHappened: 'backups.physical.monitoring.reasons.sourceUnreachable.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.sourceUnreachable.whatToDo',
  },
  [WalStreamReasonCode.WAL_CHAIN_BROKEN]: {
    whatHappened: 'backups.physical.monitoring.reasons.walChainBroken.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.walChainBroken.whatToDo',
    whatToDoWhilePending: 'backups.physical.monitoring.reasons.walChainBroken.whatToDoPending',
  },
  [WalStreamReasonCode.SLOT_MISSING]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotMissing.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotMissing.whatToDo',
    whatToDoWhilePending: 'backups.physical.monitoring.reasons.slotMissing.whatToDoPending',
  },
  [WalStreamReasonCode.SLOT_LOST]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotLost.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotLost.whatToDo',
  },
  [WalStreamReasonCode.ARCHIVE_NOT_ADVANCING]: {
    whatHappened: 'backups.physical.monitoring.reasons.archiveNotAdvancing.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.archiveNotAdvancing.whatToDo',
  },
  [WalStreamReasonCode.ARCHIVE_SLOWER_THAN_SOURCE]: {
    whatHappened: 'backups.physical.monitoring.reasons.archiveSlowerThanSource.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.archiveSlowerThanSource.whatToDo',
  },
  [WalStreamReasonCode.SLOT_KEEPS_EXTRA_WAL]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotKeepsExtraWal.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotKeepsExtraWal.whatToDo',
  },
  [WalStreamReasonCode.SLOT_LIMIT_APPROACHING]: {
    whatHappened: 'backups.physical.monitoring.reasons.slotLimitApproaching.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.slotLimitApproaching.whatToDo',
  },
  [WalStreamReasonCode.NEAR_SLOT_REBUILD]: {
    whatHappened: 'backups.physical.monitoring.reasons.nearSlotRebuild.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.nearSlotRebuild.whatToDo',
  },
  [WalStreamReasonCode.LOCAL_QUEUE_FILLING]: {
    whatHappened: 'backups.physical.monitoring.reasons.localQueueFilling.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.localQueueFilling.whatToDo',
  },
  [WalStreamReasonCode.FREQUENT_RESTARTS]: {
    whatHappened: 'backups.physical.monitoring.reasons.frequentRestarts.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.frequentRestarts.whatToDo',
  },
  [WalStreamReasonCode.FORCED_ROTATION_REFUSED]: {
    whatHappened: 'backups.physical.monitoring.reasons.forcedRotationRefused.whatHappened',
    whatToDo: 'backups.physical.monitoring.reasons.forcedRotationRefused.whatToDo',
  },
};

const NOTICE_SENTENCE_KEYS: Record<WalStreamNoticeCode, ReasonSentenceKeys> = {
  [WalStreamNoticeCode.NO_SLOT_WAL_LIMIT]: {
    whatHappened: 'backups.physical.monitoring.notices.noSlotWalLimit.whatHappened',
    whatToDo: 'backups.physical.monitoring.notices.noSlotWalLimit.whatToDo',
  },
  [WalStreamNoticeCode.SLOT_KEEPS_EXTRA_WAL_STEADILY]: {
    whatHappened: 'backups.physical.monitoring.notices.slotKeepsExtraWalSteadily.whatHappened',
    whatToDo: 'backups.physical.monitoring.notices.slotKeepsExtraWalSteadily.whatToDo',
  },
  [WalStreamNoticeCode.NO_FULL_BACKUP_YET]: {
    whatHappened: 'backups.physical.monitoring.notices.noFullBackupYet.whatHappened',
    whatToDo: 'backups.physical.monitoring.notices.noFullBackupYet.whatToDo',
  },
};

const NOTICE_STAGES: Record<WalStreamNoticeCode, PathStage> = {
  [WalStreamNoticeCode.NO_SLOT_WAL_LIMIT]: PathStage.SOURCE,
  [WalStreamNoticeCode.SLOT_KEEPS_EXTRA_WAL_STEADILY]: PathStage.SOURCE,
  [WalStreamNoticeCode.NO_FULL_BACKUP_YET]: PathStage.STORAGE,
};

const REASON_LEVEL_ORDER: Record<WalStreamReasonLevel, number> = {
  [WalStreamReasonLevel.BROKEN]: 0,
  [WalStreamReasonLevel.NEEDS_ATTENTION]: 1,
  [WalStreamReasonLevel.INFO]: 2,
};

// Broken reasons first; within a level the backend's order, which follows the documented list.
export const sortReasonsBrokenFirst = (reasons: WalStreamReason[]): WalStreamReason[] =>
  [...reasons].sort(
    (first, second) => REASON_LEVEL_ORDER[first.level] - REASON_LEVEL_ORDER[second.level],
  );

export const getReasonTexts = (
  reason: WalStreamReason,
  isFullBackupPending: boolean,
  now: Date,
): ReasonTexts => {
  const sentenceKeys = REASON_SENTENCE_KEYS[reason.code];
  const whatToDoKey =
    isFullBackupPending && sentenceKeys.whatToDoWhilePending
      ? sentenceKeys.whatToDoWhilePending
      : sentenceKeys.whatToDo;

  return {
    level: reason.level,
    stage: REASON_STAGES[reason.code],
    titleKey: WAL_STREAM_REASON_CODE_LABEL_KEYS[reason.code],
    whatHappened: { key: sentenceKeys.whatHappened, params: getReasonParams(reason, now) },
    whatToDo: { key: whatToDoKey },
  };
};

export const getNoticeTexts = (notice: WalStreamNotice): ReasonTexts => {
  const sentenceKeys = NOTICE_SENTENCE_KEYS[notice.code];

  return {
    level: notice.level,
    stage: NOTICE_STAGES[notice.code],
    titleKey: WAL_STREAM_NOTICE_CODE_LABEL_KEYS[notice.code],
    whatHappened: { key: sentenceKeys.whatHappened },
    whatToDo: { key: sentenceKeys.whatToDo },
  };
};

const getReasonParams = (reason: WalStreamReason, now: Date): MonitoringText['params'] => {
  const params: Record<string, MonitoringTextParam> = {};

  if (reason.since !== undefined) {
    params.duration = {
      kind: 'text',
      text: getRoundedDuration(getSecondsBetween(reason.since, now)),
    };
  }

  if (reason.estimatedSeconds !== undefined) {
    params.duration = { kind: 'text', text: getRoundedDuration(reason.estimatedSeconds) };
  }

  if (reason.valueBytes !== undefined) {
    params.amount = { kind: 'bytes', bytes: reason.valueBytes };
  }

  if (reason.thresholdBytes !== undefined) {
    params.limit = { kind: 'bytes', bytes: reason.thresholdBytes };
  }

  if (reason.count !== undefined) {
    params.count = { kind: 'count', count: reason.count };
  }

  return params;
};

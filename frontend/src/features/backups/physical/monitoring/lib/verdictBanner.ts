import {
  WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS,
  WalStreamHealthLevel,
} from '../../../../../entity/backups/physical';
import { MonitoringTone } from '../model/MonitoringTone';
import type { VerdictBanner } from '../model/VerdictBanner';
import { VerdictIcon } from '../model/VerdictIcon';
import { getRoundedDuration, getSecondsBetween } from './roundedDuration';

// Four missed refreshes: past this the screen no longer knows the current state, and showing the
// last verdict in its own color would claim otherwise.
export const UNCONFIRMED_STATE_AFTER_MS = 2 * 60 * 1000;

const LEVEL_TONES: Record<WalStreamHealthLevel, MonitoringTone> = {
  [WalStreamHealthLevel.HEALTHY]: MonitoringTone.SUCCESS,
  [WalStreamHealthLevel.NEEDS_ATTENTION]: MonitoringTone.WARNING,
  [WalStreamHealthLevel.BROKEN]: MonitoringTone.ERROR,
  [WalStreamHealthLevel.COLLECTING_DATA]: MonitoringTone.NEUTRAL,
  [WalStreamHealthLevel.NOT_STREAMING]: MonitoringTone.NEUTRAL,
};

const LEVEL_ICONS: Record<WalStreamHealthLevel, VerdictIcon> = {
  [WalStreamHealthLevel.HEALTHY]: VerdictIcon.CHECK,
  [WalStreamHealthLevel.NEEDS_ATTENTION]: VerdictIcon.EXCLAMATION,
  [WalStreamHealthLevel.BROKEN]: VerdictIcon.CLOSE,
  [WalStreamHealthLevel.COLLECTING_DATA]: VerdictIcon.CLOCK,
  [WalStreamHealthLevel.NOT_STREAMING]: VerdictIcon.PAUSE,
};

export const getVerdictBanner = (
  level: WalStreamHealthLevel,
  lastSuccessfulRefreshAt: Date,
  now: Date,
): VerdictBanner => {
  if (now.getTime() - lastSuccessfulRefreshAt.getTime() > UNCONFIRMED_STATE_AFTER_MS) {
    return {
      tone: MonitoringTone.NEUTRAL,
      icon: VerdictIcon.QUESTION,
      headlineKey: 'backups.physical.monitoring.banner.cannotConfirm',
      description: {
        key: 'backups.physical.monitoring.banner.cannotConfirmDescription',
        params: {
          level: { kind: 'text', text: { key: WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS[level] } },
          duration: {
            kind: 'text',
            text: getRoundedDuration(getSecondsBetween(lastSuccessfulRefreshAt, now)),
          },
        },
      },
      isConfigLinkShown: false,
      isStateConfirmed: false,
    };
  }

  return {
    tone: LEVEL_TONES[level],
    icon: LEVEL_ICONS[level],
    headlineKey: WAL_STREAM_HEALTH_LEVEL_LABEL_KEYS[level],
    description: getLevelDescription(level),
    isConfigLinkShown: level === WalStreamHealthLevel.NOT_STREAMING,
    isStateConfirmed: true,
  };
};

const getLevelDescription = (level: WalStreamHealthLevel): VerdictBanner['description'] => {
  switch (level) {
    case WalStreamHealthLevel.COLLECTING_DATA:
      return { key: 'backups.physical.monitoring.banner.collectingDataDescription' };
    case WalStreamHealthLevel.NOT_STREAMING:
      return { key: 'backups.physical.monitoring.banner.notStreamingDescription' };
    default:
      return undefined;
  }
};

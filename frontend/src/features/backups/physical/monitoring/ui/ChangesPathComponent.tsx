import { ArrowDownOutlined, ArrowRightOutlined } from '@ant-design/icons';
import { Progress, Tooltip, theme } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type WalStreamStatistics,
  WalStreamSourceState,
} from '../../../../../entity/backups/physical';
import { type TranslationKey, useLocale } from '../../../../../shared/i18n';
import { formatBytes } from '../../../../../shared/lib';
import { interpolateNumber } from '../lib/easedTexts';
import { formatBytesPerSecond } from '../lib/monitoringTextValues';
import { getAmountProgress } from '../lib/pathStages';
import { getToneColors } from '../lib/toneColors';
import type { MonitoringTone } from '../model/MonitoringTone';
import { PathStage } from '../model/PathStage';
import { useEasedTransition } from './useEasedTransition';

interface Props {
  statistics: WalStreamStatistics;
  stageTones: Record<PathStage, MonitoringTone>;
  isReducedMotion: boolean;
}

interface StageCardProps {
  titleKey: TranslationKey;
  tone: MonitoringTone;
  children: ReactNode;
}

interface AmountProps {
  labelKey: TranslationKey;
  tooltipKey: TranslationKey;
  valueBytes: number | null;
  limitBytes?: number;
  isReducedMotion: boolean;
}

interface ArrowProps {
  labelKey: TranslationKey;
  tooltipKey: TranslationKey;
  bytesPerSecond: number | null;
}

const StageCardComponent = ({ titleKey, tone, children }: StageCardProps) => {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  return (
    <div
      className="flex-1 rounded-lg border-2 bg-white p-3 dark:bg-gray-800"
      style={{ borderColor: getToneColors(token, tone).border }}
    >
      <div className="mb-2 font-semibold">{t(titleKey)}</div>
      <div className="space-y-2 text-sm">{children}</div>
    </div>
  );
};

const easeAmount = (fromBytes: number, toBytes: number, progress: number) =>
  interpolateNumber(fromBytes, toBytes, progress);

const AmountComponent = ({
  labelKey,
  tooltipKey,
  valueBytes,
  limitBytes,
  isReducedMotion,
}: AmountProps) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  const easedBytes = useEasedTransition(valueBytes ?? 0, easeAmount, isReducedMotion);

  const progress =
    valueBytes !== null && limitBytes !== undefined
      ? getAmountProgress(valueBytes, limitBytes)
      : null;

  return (
    <div>
      <Tooltip title={t(tooltipKey)}>
        <span tabIndex={0} className="cursor-help border-b border-dotted border-gray-400">
          {t(labelKey)}
        </span>
      </Tooltip>

      <div className="font-semibold">
        {valueBytes === null
          ? t('backups.physical.monitoring.path.unknownAmount')
          : formatBytes(easedBytes, formatNumber)}
      </div>

      {progress && (
        <>
          <Progress
            percent={progress.percent}
            showInfo={false}
            size="small"
            status={progress.isOverLimit ? 'exception' : 'normal'}
          />
          {progress.isOverLimit && (
            <div className="text-xs font-semibold text-red-600 dark:text-red-400">
              {t('backups.physical.monitoring.path.overLimit')}
            </div>
          )}
        </>
      )}
    </div>
  );
};

const ArrowComponent = ({ labelKey, tooltipKey, bytesPerSecond }: ArrowProps) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  const rate =
    bytesPerSecond === null
      ? t('backups.physical.monitoring.path.unknownAmount')
      : formatBytesPerSecond(bytesPerSecond, t, formatNumber);

  return (
    <div className="flex flex-col items-center justify-center px-2 text-xs text-gray-600 dark:text-gray-400">
      <Tooltip title={t(tooltipKey)}>
        <span tabIndex={0} className="cursor-help text-center">
          {t(labelKey, { rate })}
        </span>
      </Tooltip>
      <ArrowRightOutlined className="hidden text-lg md:block" aria-hidden />
      <ArrowDownOutlined className="text-lg md:hidden" aria-hidden />
    </div>
  );
};

export const ChangesPathComponent = ({ statistics, stageTones, isReducedMotion }: Props) => {
  const { t } = useTranslation();
  const { formatNumber, formatRelativeTime } = useLocale();

  const { current, keepingUp } = statistics;
  const isSourceReachable = current?.sourceState === WalStreamSourceState.REACHABLE;
  const slotRebuildLagBytes = keepingUp.slotRebuildLagBytes;

  return (
    <div className="mt-4 flex flex-col items-stretch md:flex-row">
      <StageCardComponent
        titleKey="backups.physical.monitoring.path.database"
        tone={stageTones[PathStage.SOURCE]}
      >
        <AmountComponent
          labelKey="backups.physical.monitoring.path.notReceivedYet"
          tooltipKey="backups.physical.monitoring.tooltips.notReceivedYet"
          valueBytes={current?.slotLagBytes ?? null}
          limitBytes={slotRebuildLagBytes > 0 ? slotRebuildLagBytes : undefined}
          isReducedMotion={isReducedMotion}
        />

        {slotRebuildLagBytes > 0 && (
          <Tooltip title={t('backups.physical.monitoring.tooltips.rebuildLimit')}>
            <div tabIndex={0} className="cursor-help text-xs text-gray-600 dark:text-gray-400">
              {t('backups.physical.monitoring.path.rebuildLimit', {
                amount: formatBytes(slotRebuildLagBytes, formatNumber),
              })}
            </div>
          </Tooltip>
        )}

        {isSourceReachable && (
          <Tooltip title={t('backups.physical.monitoring.tooltips.allowanceLeft')}>
            <div tabIndex={0} className="cursor-help text-xs text-gray-600 dark:text-gray-400">
              {current?.slotSafeWalSizeBytes != null
                ? t('backups.physical.monitoring.path.allowanceLeft', {
                    amount: formatBytes(current.slotSafeWalSizeBytes, formatNumber),
                  })
                : t('backups.physical.monitoring.path.noAllowanceLimit')}
            </div>
          </Tooltip>
        )}
      </StageCardComponent>

      <ArrowComponent
        labelKey="backups.physical.monitoring.path.writeRate"
        tooltipKey="backups.physical.monitoring.tooltips.writeRate"
        bytesPerSecond={keepingUp.sourceWriteBytesPerSecond}
      />

      <StageCardComponent
        titleKey="backups.physical.monitoring.path.databasus"
        tone={stageTones[PathStage.DATABASUS]}
      >
        <AmountComponent
          labelKey="backups.physical.monitoring.path.waitingInBuffer"
          tooltipKey="backups.physical.monitoring.tooltips.waitingInBuffer"
          valueBytes={current?.localQueueBytes ?? null}
          limitBytes={current?.localQueuePauseBytes}
          isReducedMotion={isReducedMotion}
        />

        {current && (
          <Tooltip title={t('backups.physical.monitoring.tooltips.pauseLimit')}>
            <div tabIndex={0} className="cursor-help text-xs text-gray-600 dark:text-gray-400">
              {t('backups.physical.monitoring.path.pauseLimit', {
                amount: formatBytes(current.localQueuePauseBytes, formatNumber),
              })}
            </div>
          </Tooltip>
        )}

        {current && !current.isReceiverRunning && (
          <div className="text-xs font-semibold">
            {t('backups.physical.monitoring.path.receivingPaused')}
          </div>
        )}
      </StageCardComponent>

      <ArrowComponent
        labelKey="backups.physical.monitoring.path.saveRate"
        tooltipKey="backups.physical.monitoring.tooltips.saveRate"
        bytesPerSecond={keepingUp.archiveBytesPerSecond}
      />

      <StageCardComponent
        titleKey="backups.physical.monitoring.path.storage"
        tone={stageTones[PathStage.STORAGE]}
      >
        <Tooltip title={t('backups.physical.monitoring.tooltips.lastSaved')}>
          <div tabIndex={0} className="cursor-help">
            {current?.lastArchivedWalAt
              ? t('backups.physical.monitoring.path.lastSaved', {
                  relativeTime: formatRelativeTime(current.lastArchivedWalAt),
                })
              : t('backups.physical.monitoring.path.nothingSavedYet')}
          </div>
        </Tooltip>
      </StageCardComponent>
    </div>
  );
};

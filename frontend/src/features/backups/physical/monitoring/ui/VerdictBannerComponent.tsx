import {
  CheckCircleFilled,
  ClockCircleOutlined,
  CloseCircleFilled,
  ExclamationCircleFilled,
  PauseCircleFilled,
  QuestionCircleFilled,
} from '@ant-design/icons';
import { theme } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { translateLocalizedText } from '../../../../../shared/i18n';
import { interpolateMonitoringTexts } from '../lib/easedTexts';
import { getMeasurementAge, getSecondsBetween } from '../lib/roundedDuration';
import { getToneColors } from '../lib/toneColors';
import { UNCONFIRMED_STATE_AFTER_MS } from '../lib/verdictBanner';
import type { MonitoringText } from '../model/MonitoringText';
import type { VerdictBanner } from '../model/VerdictBanner';
import { VerdictIcon } from '../model/VerdictIcon';
import { MonitoringTextComponent } from './MonitoringTextComponent';
import { useEasedTransition } from './useEasedTransition';

interface Props {
  banner: VerdictBanner;
  keepingUpSentences: MonitoringText[];
  sampledAt: string | null;
  now: Date;
  isReducedMotion: boolean;
  onOpenConfig: () => void;
}

const VERDICT_ICONS: Record<VerdictIcon, ReactNode> = {
  [VerdictIcon.CHECK]: <CheckCircleFilled />,
  [VerdictIcon.EXCLAMATION]: <ExclamationCircleFilled />,
  [VerdictIcon.CLOSE]: <CloseCircleFilled />,
  [VerdictIcon.CLOCK]: <ClockCircleOutlined />,
  [VerdictIcon.PAUSE]: <PauseCircleFilled />,
  [VerdictIcon.QUESTION]: <QuestionCircleFilled />,
};

export const VerdictBannerComponent = ({
  banner,
  keepingUpSentences,
  sampledAt,
  now,
  isReducedMotion,
  onOpenConfig,
}: Props) => {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const easedSentences = useEasedTransition(
    keepingUpSentences,
    interpolateMonitoringTexts,
    isReducedMotion,
  );

  const toneColors = getToneColors(token, banner.tone);
  const isLive =
    sampledAt !== null &&
    now.getTime() - new Date(sampledAt).getTime() < UNCONFIRMED_STATE_AFTER_MS;

  return (
    <div
      className="w-full rounded-lg border p-4"
      style={{
        backgroundColor: toneColors.background,
        borderColor: toneColors.border,
        transition: isReducedMotion ? undefined : 'background-color 300ms, border-color 300ms',
      }}
    >
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="flex items-start gap-3">
          <div className="text-3xl leading-none" style={{ color: toneColors.icon }} aria-hidden>
            {VERDICT_ICONS[banner.icon]}
          </div>

          <div>
            <div aria-live="polite" className="text-lg font-semibold">
              {t(banner.headlineKey)}
            </div>

            {banner.description && (
              <div className="mt-1">
                <MonitoringTextComponent text={banner.description} onOpenConfig={onOpenConfig} />
              </div>
            )}

            {easedSentences.length > 0 && (
              <div className="mt-1 text-gray-700 dark:text-gray-300">
                {easedSentences.map((sentence) => (
                  <span key={sentence.key} className="mr-1">
                    <MonitoringTextComponent text={sentence} />
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
          <span
            aria-hidden
            className={`inline-block h-2.5 w-2.5 rounded-full ${isLive && !isReducedMotion ? 'animate-pulse' : ''}`}
            style={{ backgroundColor: isLive ? token.colorSuccess : token.colorTextQuaternary }}
          />

          {sampledAt
            ? t('backups.physical.monitoring.banner.measuredAgo', {
                duration: translateLocalizedText(
                  getMeasurementAge(getSecondsBetween(sampledAt, now)),
                  t,
                ),
              })
            : t('backups.physical.monitoring.banner.notMeasuredYet')}
        </div>
      </div>
    </div>
  );
};

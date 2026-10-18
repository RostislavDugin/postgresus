import { Alert, Button, Skeleton } from 'antd';
import { type ReactNode, memo, useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  WalStreamHealthLevel,
  WalStreamStatisticsPeriod,
  physicalWalStreamStatisticsApi,
} from '../../../../../entity/backups/physical';
import type { Database } from '../../../../../entity/databases';
import { translateApiError } from '../../../../../shared/i18n';
import { getKeepingUpSentences } from '../lib/keepingUpSentences';
import { getStageTones } from '../lib/pathStages';
import { getNoticeTexts, getReasonTexts, sortReasonsBrokenFirst } from '../lib/reasonTexts';
import { getVerdictBanner } from '../lib/verdictBanner';
import {
  type PageVisibility,
  type RefreshTimer,
  WalStreamRefreshController,
} from '../lib/walStreamRefreshController';
import type { WalStreamRefreshState } from '../model/WalStreamRefreshState';
import { ChangesPathComponent } from './ChangesPathComponent';
import { MonitoringReasonsComponent } from './MonitoringReasonsComponent';
import { useCurrentTime } from './useCurrentTime';
import { usePrefersReducedMotion } from './usePrefersReducedMotion';
import { VerdictBannerComponent } from './VerdictBannerComponent';
import { WalStreamChartsComponent } from './WalStreamChartsComponent';
import { WalStreamDetailsComponent } from './WalStreamDetailsComponent';

interface Props {
  database: Database;
  isDirectlyUnderTab: boolean;
  onOpenConfig: () => void;
}

// eslint-disable-next-line i18next/no-literal-string -- DOM event name
const VISIBILITY_CHANGE_EVENT = 'visibilitychange';

const browserPageVisibility: PageVisibility = {
  isVisible: () => document.visibilityState !== 'hidden',
  subscribe: (onVisibilityChange) => {
    document.addEventListener(VISIBILITY_CHANGE_EVENT, onVisibilityChange);

    return () => document.removeEventListener(VISIBILITY_CHANGE_EVENT, onVisibilityChange);
  },
};

const browserRefreshTimer: RefreshTimer = {
  setInterval: (callback, intervalMs) => window.setInterval(callback, intervalMs),
  clearInterval: (handle) => window.clearInterval(handle as number),
};

// Longer than any chart's entry animation, so the animation of a new period finishes before
// later refreshes stop animating.
const CHART_ENTRY_ANIMATION_MS = 1500;

// The charts re-render only when a refresh brings a new series, not on every tick of the clock.
const MemoizedWalStreamChartsComponent = memo(WalStreamChartsComponent);

const MonitoringContainerComponent = ({
  isDirectlyUnderTab,
  children,
}: {
  isDirectlyUnderTab: boolean;
  children: ReactNode;
}) => (
  <div
    className={`w-full bg-white p-3 shadow md:p-5 dark:bg-gray-800 ${isDirectlyUnderTab ? 'rounded-tr-md rounded-br-md rounded-bl-md' : 'rounded-md'}`}
  >
    {children}
  </div>
);

const MonitoringSkeletonComponent = () => (
  <div className="space-y-4">
    <div className="h-24 rounded-lg border border-gray-200 p-4 dark:border-gray-700">
      <Skeleton active title paragraph={{ rows: 1 }} />
    </div>
    <div className="h-36 rounded-lg border border-gray-200 p-4 dark:border-gray-700">
      <Skeleton active paragraph={{ rows: 3 }} />
    </div>
    <div className="h-[760px] rounded-lg border border-gray-200 p-4 dark:border-gray-700">
      <Skeleton active paragraph={{ rows: 12 }} />
    </div>
  </div>
);

export const WalStreamMonitoringComponent = ({
  database,
  isDirectlyUnderTab,
  onOpenConfig,
}: Props) => {
  const { t } = useTranslation();

  const [period, setPeriod] = useState(WalStreamStatisticsPeriod.HOUR);
  const [refreshState, setRefreshState] = useState<WalStreamRefreshState | null>(null);
  const [lastAnimatedPeriod, setLastAnimatedPeriod] = useState<WalStreamStatisticsPeriod | null>(
    null,
  );

  const controllerRef = useRef<WalStreamRefreshController | null>(null);

  const retryFirstLoad = () => {
    controllerRef.current?.retry();
  };

  const now = useCurrentTime();
  const isReducedMotion = usePrefersReducedMotion();

  const changePeriod = useCallback((newPeriod: WalStreamStatisticsPeriod) => {
    setPeriod(newPeriod);
    controllerRef.current?.setPeriod(newPeriod);
  }, []);

  useEffect(() => {
    const controller = new WalStreamRefreshController(
      {
        fetchStatistics: (requestedPeriod) =>
          physicalWalStreamStatisticsApi.getWalStreamStatistics(database.id, requestedPeriod),
        timer: browserRefreshTimer,
        pageVisibility: browserPageVisibility,
        getNow: () => new Date(),
      },
      period,
      setRefreshState,
    );

    controllerRef.current = controller;
    controller.start();

    return () => controller.stop();
  }, [database.id]);

  const shownSeriesPeriod = refreshState?.statistics?.series.period ?? null;

  useEffect(() => {
    if (shownSeriesPeriod === null || shownSeriesPeriod === lastAnimatedPeriod) {
      return;
    }

    const timeout = setTimeout(
      () => setLastAnimatedPeriod(shownSeriesPeriod),
      CHART_ENTRY_ANIMATION_MS,
    );

    return () => clearTimeout(timeout);
  }, [shownSeriesPeriod, lastAnimatedPeriod]);

  const statistics = refreshState?.statistics ?? null;

  if (!statistics || !refreshState?.lastSuccessfulRefreshAt) {
    return (
      <MonitoringContainerComponent isDirectlyUnderTab={isDirectlyUnderTab}>
        {refreshState?.firstLoadError ? (
          <Alert
            type="error"
            showIcon
            message={translateApiError(refreshState.firstLoadError, t)}
            action={
              <Button size="small" onClick={retryFirstLoad}>
                {t('common.actions.retry')}
              </Button>
            }
          />
        ) : (
          <MonitoringSkeletonComponent />
        )}
      </MonitoringContainerComponent>
    );
  }

  const { health } = statistics;
  const banner = getVerdictBanner(health.level, refreshState.lastSuccessfulRefreshAt, now);
  const isVerdictAboutStreaming =
    health.level !== WalStreamHealthLevel.NOT_STREAMING &&
    health.level !== WalStreamHealthLevel.COLLECTING_DATA;

  const reasons = sortReasonsBrokenFirst(health.reasons).map((reason) =>
    getReasonTexts(reason, health.isFullBackupPending, now),
  );
  const notices = health.notices.map((notice) => ({
    code: notice.code,
    texts: getNoticeTexts(notice),
  }));

  return (
    <MonitoringContainerComponent isDirectlyUnderTab={isDirectlyUnderTab}>
      <VerdictBannerComponent
        banner={banner}
        keepingUpSentences={
          banner.isStateConfirmed && isVerdictAboutStreaming
            ? getKeepingUpSentences(statistics.keepingUp)
            : []
        }
        sampledAt={statistics.current?.sampledAt ?? null}
        now={now}
        isReducedMotion={isReducedMotion}
        onOpenConfig={onOpenConfig}
      />

      {refreshState.isLastRefreshFailed && (
        <Alert
          className="mt-3"
          type="warning"
          showIcon
          message={t('backups.physical.monitoring.banner.refreshFailed')}
        />
      )}

      <MonitoringReasonsComponent reasons={reasons} notices={notices} />

      <ChangesPathComponent
        statistics={statistics}
        stageTones={getStageTones(health.level, health.reasons)}
        isReducedMotion={isReducedMotion}
      />

      <MemoizedWalStreamChartsComponent
        series={statistics.series}
        period={period}
        onPeriodChange={changePeriod}
        isAnimationActive={!isReducedMotion && statistics.series.period !== lastAnimatedPeriod}
      />

      <WalStreamDetailsComponent statistics={statistics} />
    </MonitoringContainerComponent>
  );
};

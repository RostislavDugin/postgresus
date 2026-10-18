import { Segmented, theme } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Scatter,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import {
  WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS,
  type WalStreamSeries,
  WalStreamStatisticsPeriod,
} from '../../../../../entity/backups/physical';
import { type TranslationKey, useLocale } from '../../../../../shared/i18n';
import { formatBytes } from '../../../../../shared/lib';
import { useTheme } from '../../../../../shared/theme';
import { getUserTimeFormat } from '../../../../../shared/time';
import { WAL_STREAM_CHART_COLORS } from '../lib/chartColors';
import { getDataSinceMs, hasAnyChartValues, toChartRows } from '../lib/chartRows';
import { formatBytesPerSecond } from '../lib/monitoringTextValues';
import type { WalStreamChartRow } from '../model/WalStreamChartRow';

interface Props {
  series: WalStreamSeries;
  period: WalStreamStatisticsPeriod;
  onPeriodChange: (period: WalStreamStatisticsPeriod) => void;
  isAnimationActive: boolean;
}

interface ChartLine {
  dataKey: keyof WalStreamChartRow;
  labelKey: TranslationKey;
  color: string;
  isDashed: boolean;
}

interface ChartFrameProps {
  titleKey: TranslationKey;
  subtitleKey?: TranslationKey;
  rows: WalStreamChartRow[];
  series: WalStreamSeries;
  lines: ChartLine[];
  isRate: boolean;
  isAnimationActive: boolean;
  pauseLimitBytes?: number;
}

const CHART_HEIGHT_PX = 220;

// eslint-disable-next-line i18next/no-literal-string -- recharts sync group id
const CHARTS_SYNC_ID = 'walStreamMonitoring';

const PERIODS = [
  WalStreamStatisticsPeriod.HOUR,
  WalStreamStatisticsPeriod.DAY,
  WalStreamStatisticsPeriod.WEEK,
];

/* eslint-disable i18next/no-literal-string -- dayjs format tokens, not copy */
const getTimeTickFormat = (period: WalStreamStatisticsPeriod): string => {
  const clockFormat = getUserTimeFormat().use12Hours ? 'h:mm A' : 'HH:mm';

  return period === WalStreamStatisticsPeriod.WEEK ? `ddd ${clockFormat}` : clockFormat;
};
/* eslint-enable i18next/no-literal-string */

const ChartFrameComponent = ({
  titleKey,
  subtitleKey,
  rows,
  series,
  lines,
  isRate,
  isAnimationActive,
  pauseLimitBytes,
}: ChartFrameProps) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();
  const { token } = theme.useToken();
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];
  const timeTickFormat = getTimeTickFormat(series.period);

  const formatAmount = (value: number) =>
    isRate ? formatBytesPerSecond(value, t, formatNumber) : formatBytes(value, formatNumber);

  return (
    <div>
      <div className="font-semibold">{t(titleKey)}</div>
      {subtitleKey && (
        <div className="text-xs text-gray-600 dark:text-gray-400">{t(subtitleKey)}</div>
      )}

      <div style={{ width: '100%', height: CHART_HEIGHT_PX }}>
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart
            data={rows}
            syncId={CHARTS_SYNC_ID}
            margin={{ top: 8, right: 16, left: 8, bottom: 0 }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke={token.colorBorderSecondary} />
            <XAxis
              dataKey="timestampMs"
              type="number"
              scale="time"
              domain={[new Date(series.from).getTime(), new Date(series.to).getTime()]}
              tickFormatter={(timestampMs: number) => dayjs(timestampMs).format(timeTickFormat)}
              tick={{ fontSize: 11, fill: token.colorTextSecondary }}
              stroke={token.colorBorder}
            />
            <YAxis
              width={80}
              tickFormatter={formatAmount}
              tick={{ fontSize: 11, fill: token.colorTextSecondary }}
              stroke={token.colorBorder}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: token.colorBgElevated,
                borderColor: token.colorBorder,
                color: token.colorText,
              }}
              labelFormatter={(timestampMs) => dayjs(Number(timestampMs)).format(timeTickFormat)}
              formatter={(value) => (typeof value === 'number' ? formatAmount(value) : value)}
            />
            <Legend verticalAlign="top" height={28} wrapperStyle={{ fontSize: 12 }} />

            {lines.map((line) => (
              <Line
                key={line.dataKey}
                type="monotone"
                dataKey={line.dataKey}
                name={t(line.labelKey)}
                stroke={line.color}
                strokeWidth={2}
                strokeDasharray={line.isDashed ? '6 4' : undefined}
                dot={false}
                connectNulls={false}
                isAnimationActive={isAnimationActive}
              />
            ))}

            {pauseLimitBytes !== undefined && (
              <ReferenceLine
                y={pauseLimitBytes}
                stroke={chartColors.pauseLimit}
                strokeDasharray="4 4"
                label={{
                  value: t('backups.physical.monitoring.charts.pauseLimit'),
                  position: 'insideTopRight',
                  fill: token.colorTextSecondary,
                  fontSize: 11,
                }}
              />
            )}

            {pauseLimitBytes !== undefined && (
              <Scatter
                dataKey="restartMarkerBytes"
                name={t('backups.physical.monitoring.charts.restarts')}
                fill={chartColors.restartMarker}
                isAnimationActive={isAnimationActive}
              />
            )}

            {pauseLimitBytes !== undefined && (
              <Scatter
                dataKey="pauseMarkerBytes"
                name={t('backups.physical.monitoring.charts.pauses')}
                fill={chartColors.pauseMarker}
                shape="diamond"
                isAnimationActive={isAnimationActive}
              />
            )}
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};

export const WalStreamChartsComponent = ({
  series,
  period,
  onPeriodChange,
  isAnimationActive,
}: Props) => {
  const { t } = useTranslation();
  const { resolvedTheme } = useTheme();

  const chartColors = WAL_STREAM_CHART_COLORS[resolvedTheme];
  const rows = toChartRows(series.points);
  const dataSinceMs = getDataSinceMs(series);
  const latestPauseLimitBytes = [...series.points]
    .reverse()
    .find((point) => point.localQueuePauseBytes !== null)?.localQueuePauseBytes;

  return (
    <div className="mt-6">
      <Segmented
        value={period}
        onChange={(value) => onPeriodChange(value as WalStreamStatisticsPeriod)}
        options={PERIODS.map((periodOption) => ({
          value: periodOption,
          label: t(WAL_STREAM_STATISTICS_PERIOD_LABEL_KEYS[periodOption]),
        }))}
      />

      {!hasAnyChartValues(series) ? (
        <div className="mt-4 rounded-lg border border-dashed border-gray-300 p-6 text-center text-gray-600 dark:border-gray-600 dark:text-gray-400">
          {t('backups.physical.monitoring.charts.noHistory')}
        </div>
      ) : (
        <div className="mt-4 space-y-6">
          {dataSinceMs !== null && (
            <div className="text-xs text-gray-600 dark:text-gray-400">
              {t('backups.physical.monitoring.charts.dataSince', {
                time: dayjs(dataSinceMs).format(getTimeTickFormat(series.period)),
              })}
            </div>
          )}

          <ChartFrameComponent
            titleKey="backups.physical.monitoring.charts.keepingUpTitle"
            rows={rows}
            series={series}
            isRate
            isAnimationActive={isAnimationActive}
            lines={[
              {
                dataKey: 'sourceWriteBytesPerSecond',
                labelKey: 'backups.physical.monitoring.charts.databaseWrites',
                color: chartColors.sourceWrites,
                isDashed: false,
              },
              {
                dataKey: 'archiveBytesPerSecond',
                labelKey: 'backups.physical.monitoring.charts.savedToStorage',
                color: chartColors.archive,
                isDashed: true,
              },
            ]}
          />

          <ChartFrameComponent
            titleKey="backups.physical.monitoring.charts.unsavedTitle"
            subtitleKey="backups.physical.monitoring.charts.unsavedSubtitle"
            rows={rows}
            series={series}
            isRate={false}
            isAnimationActive={isAnimationActive}
            lines={[
              {
                dataKey: 'maxUnarchivedWalBytes',
                labelKey: 'backups.physical.monitoring.charts.notSavedYet',
                color: chartColors.notSavedYet,
                isDashed: false,
              },
              {
                dataKey: 'maxSlotLagBytes',
                labelKey: 'backups.physical.monitoring.charts.notReceivedYet',
                color: chartColors.notReceivedYet,
                isDashed: true,
              },
            ]}
          />

          <ChartFrameComponent
            titleKey="backups.physical.monitoring.charts.bufferTitle"
            rows={rows}
            series={series}
            isRate={false}
            isAnimationActive={isAnimationActive}
            pauseLimitBytes={latestPauseLimitBytes ?? 0}
            lines={[
              {
                dataKey: 'maxLocalQueueBytes',
                labelKey: 'backups.physical.monitoring.charts.waitingInBuffer',
                color: chartColors.buffer,
                isDashed: false,
              },
            ]}
          />
        </div>
      )}
    </div>
  );
};

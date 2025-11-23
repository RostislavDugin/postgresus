import { Select, Spin, Tooltip } from 'antd';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';

import type { Database } from '../../../entity/databases';
import { HealthStatus } from '../../../entity/databases/model/HealthStatus';
import {
  type HealthcheckAttempt,
  healthcheckAttemptApi,
  healthcheckConfigApi,
} from '../../../entity/healthcheck';
import { getUserShortTimeFormat } from '../../../shared/time/getUserTimeFormat';

interface Props {
  database: Database;
}

let lastLoadTime = 0;

const getAfterDateByPeriod = (period: 'today' | '7d' | '30d' | 'all'): Date => {
  const afterDate = new Date();

  if (period === 'today') {
    return new Date(afterDate.setDate(afterDate.getDate() - 1));
  }

  if (period === '7d') {
    return new Date(afterDate.setDate(afterDate.getDate() - 7));
  }

  if (period === '30d') {
    return new Date(afterDate.setDate(afterDate.getDate() - 30));
  }

  if (period === 'all') {
    return new Date(0);
  }

  return afterDate;
};

export const HealthckeckAttemptsComponent = ({ database }: Props) => {
  const [isHealthcheckConfigLoading, setIsHealthcheckConfigLoading] = useState(false);
  const [isShowHealthcheckConfig, setIsShowHealthcheckConfig] = useState(false);

  const [isLoading, setIsLoading] = useState(false);
  const [healthcheckAttempts, setHealthcheckAttempts] = useState<HealthcheckAttempt[]>([]);
  const [period, setPeriod] = useState<'today' | '7d' | '30d' | 'all'>('today');

  const loadHealthcheckAttempts = async (isShowLoading = true) => {
    if (isShowLoading) {
      setIsLoading(true);
    }

    try {
      const currentTime = Date.now();
      lastLoadTime = currentTime;

      const afterDate = getAfterDateByPeriod(period);

      const healthcheckAttempts = await healthcheckAttemptApi.getAttemptsByDatabase(
        database.id,
        afterDate,
      );

      if (currentTime != lastLoadTime) {
        return;
      }

      setHealthcheckAttempts(healthcheckAttempts);
    } catch (e) {
      alert((e as Error).message);
    }

    if (isShowLoading) {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    let interval: number | null = null;

    setIsHealthcheckConfigLoading(true);
    healthcheckConfigApi.getHealthcheckConfig(database.id).then((healthcheckConfig) => {
      setIsHealthcheckConfigLoading(false);

      if (healthcheckConfig.isHealthcheckEnabled) {
        setIsShowHealthcheckConfig(true);
        loadHealthcheckAttempts();

        // Set up interval only if healthcheck
        // is enabled and period is 'today'
        if (period === 'today') {
          interval = setInterval(() => {
            loadHealthcheckAttempts(false);
          }, 60_000); // 5 seconds
        }
      }
    });

    return () => {
      if (interval) {
        clearInterval(interval);
      }
    };
  }, [period]);

  if (isHealthcheckConfigLoading) {
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );
  }

  if (!isShowHealthcheckConfig) {
    return <div />;
  }

  return (
    <div className="w-full rounded-tr-md rounded-br-md rounded-bl-md bg-white p-3 shadow sm:p-5">
      <h2 className="text-lg font-bold sm:text-xl">Healthcheck attempts</h2>

      <div className="mt-3 flex flex-col gap-2 sm:mt-4 sm:flex-row sm:items-center">
        <span className="text-sm font-medium sm:mr-2">Period</span>
        <Select
          size="small"
          value={period}
          onChange={(value) => setPeriod(value)}
          className="w-full sm:w-[120px]"
          options={[
            { value: 'today', label: 'Today' },
            { value: '7d', label: '7 days' },
            { value: '30d', label: '30 days' },
            { value: 'all', label: 'All time' },
          ]}
        />
      </div>

      <div className="mt-4 sm:mt-5" />

      {isLoading ? (
        <div className="flex justify-center">
          <Spin size="small" />
        </div>
      ) : (
        <div className="flex flex-wrap gap-1">
          {healthcheckAttempts.length > 0 ? (
            healthcheckAttempts.map((healthcheckAttempt) => (
              <Tooltip
                key={healthcheckAttempt.createdAt.toString()}
                title={`${dayjs(healthcheckAttempt.createdAt).format(getUserShortTimeFormat().format)} (${dayjs(healthcheckAttempt.createdAt).fromNow()})`}
              >
                <div
                  className={`h-[8px] w-[8px] cursor-pointer rounded-[2px] ${
                    healthcheckAttempt.status === HealthStatus.AVAILABLE
                      ? 'bg-green-500'
                      : 'bg-red-500'
                  }`}
                />
              </Tooltip>
            ))
          ) : (
            <div className="text-xs text-gray-400">No data yet</div>
          )}
        </div>
      )}
    </div>
  );
};

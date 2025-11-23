import { Spin } from 'antd';
import { useEffect, useState } from 'react';

import { healthcheckConfigApi } from '../../../entity/healthcheck';
import type { HealthcheckConfig } from '../../../entity/healthcheck';

interface Props {
  databaseId: string;
}

export const ShowHealthcheckConfigComponent = ({ databaseId }: Props) => {
  const [isLoading, setIsLoading] = useState(false);
  const [healthcheckConfig, setHealthcheckConfig] = useState<HealthcheckConfig | undefined>(
    undefined,
  );

  useEffect(() => {
    setIsLoading(true);
    healthcheckConfigApi
      .getHealthcheckConfig(databaseId)
      .then((config) => {
        setHealthcheckConfig(config);
      })
      .catch((error) => {
        alert(error.message);
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [databaseId]);

  if (isLoading) {
    return <Spin size="small" />;
  }

  if (!healthcheckConfig) {
    return <div />;
  }

  return (
    <div className="space-y-4">
      <div className="mb-1 flex items-center">
        <div className="min-w-[180px]">Is health check enabled</div>
        <div>{healthcheckConfig.isHealthcheckEnabled ? 'Yes' : 'No'}</div>
      </div>

      {healthcheckConfig.isHealthcheckEnabled && (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">Notify when unavailable</div>
            <div className="lg:w-[200px]">
              {healthcheckConfig.isSentNotificationWhenUnavailable ? 'Yes' : 'No'}
            </div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">Check interval (minutes)</div>
            <div className="lg:w-[200px]">{healthcheckConfig.intervalMinutes}</div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">Attempts before down</div>
            <div className="lg:w-[200px]">{healthcheckConfig.attemptsBeforeConcideredAsDown}</div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">Store attempts (days)</div>
            <div className="lg:w-[200px]">{healthcheckConfig.storeAttemptsDays}</div>
          </div>
        </>
      )}
    </div>
  );
};

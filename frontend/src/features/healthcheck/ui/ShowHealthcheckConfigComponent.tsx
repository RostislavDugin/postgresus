import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { healthcheckConfigApi } from '../../../entity/healthcheck';
import type { HealthcheckConfig } from '../../../entity/healthcheck';

interface Props {
  databaseId: string;
}

export const ShowHealthcheckConfigComponent = ({ databaseId }: Props) => {
  const { t } = useTranslation(['healthcheck', 'common']);
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
        <div className="min-w-[180px]">{t('healthcheck:show.is_enabled_label')}</div>
        <div className="w-[250px]">
          {healthcheckConfig.isHealthcheckEnabled ? t('common:common.yes') : t('common:common.no')}
        </div>
      </div>

      {healthcheckConfig.isHealthcheckEnabled && (
        <>
          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">{t('healthcheck:show.notify_when_unavailable_label')}</div>
            <div className="w-[250px]">
              {healthcheckConfig.isSentNotificationWhenUnavailable ? t('common:common.yes') : t('common:common.no')}
            </div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">{t('healthcheck:show.check_interval_label')}</div>
            <div className="w-[250px]">{healthcheckConfig.intervalMinutes}</div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">{t('healthcheck:show.attempts_before_down_label')}</div>
            <div className="w-[250px]">{healthcheckConfig.attemptsBeforeConcideredAsDown}</div>
          </div>

          <div className="mb-1 flex items-center">
            <div className="min-w-[180px]">{t('healthcheck:show.store_attempts_label')}</div>
            <div className="w-[250px]">{healthcheckConfig.storeAttemptsDays}</div>
          </div>
        </>
      )}
    </div>
  );
};

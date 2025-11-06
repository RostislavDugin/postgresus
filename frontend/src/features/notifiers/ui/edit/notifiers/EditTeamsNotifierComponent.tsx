import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditTeamsNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  const { t } = useTranslation(['notifier']);
  const value = notifier?.teamsNotifier?.powerAutomateUrl || '';

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const powerAutomateUrl = e.target.value.trim();
    setNotifier({
      ...notifier,
      teamsNotifier: {
        ...(notifier.teamsNotifier ?? {}),
        powerAutomateUrl,
      },
    });
    setIsUnsaved(true);
  };

  return (
    <>
      <div className="mb-1 ml-[130px] max-w-[200px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-blue-600"
          href="https://postgresus.com/notifier-teams"
          target="_blank"
          rel="noreferrer"
        >
          {t('notifier:form.teams_link')}
        </a>
      </div>

      <div className="flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.teams_power_automate_url_label')}</div>

        <div className="w-[250px]">
          <Input
            value={value}
            onChange={onChange}
            size="small"
            className="w-full"
            placeholder={t('notifier:form.teams_power_automate_url_placeholder')}
          />
        </div>

        <Tooltip
          className="cursor-pointer"
          title={t('notifier:form.teams_power_automate_url_tooltip')}
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>
    </>
  );
}

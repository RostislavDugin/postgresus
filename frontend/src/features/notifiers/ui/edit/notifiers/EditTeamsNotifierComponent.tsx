import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';
import React from 'react';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditTeamsNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
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
    setUnsaved();
  };

  return (
    <>
      <div className="mb-1 max-w-[250px] sm:ml-[150px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-blue-600"
          href="https://postgresus.com/notifiers/teams"
          target="_blank"
          rel="noreferrer"
        >
          How to connect Microsoft Teams?
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Power Automate URL</div>
        <div className="flex items-center">
          <Input
            value={value}
            onChange={onChange}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="https://prod-00.westeurope.logic.azure.com:443/workflows/....."
          />

          <Tooltip
            className="cursor-pointer"
            title="HTTP endpoint from your Power Automate flow (When an HTTP request is received)"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>
    </>
  );
}

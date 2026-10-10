import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';
import {
  AdvancedFieldRowComponent,
  AdvancedSettingsToggleComponent,
} from '../../../../../shared/ui';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

const chatIdBotLink = <a href="https://t.me/getmyid_bot" target="_blank" rel="noreferrer" />;

export function EditTelegramNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  const { t } = useTranslation();
  const [isShowHowToGetChatId, setIsShowHowToGetChatId] = useState(false);
  const [isShowAdvanced, setIsShowAdvanced] = useState(
    !!notifier.telegramNotifier?.botApiUrl || !!notifier.telegramNotifier?.isProxyEnabled,
  );

  useEffect(() => {
    if (notifier.telegramNotifier?.threadId && !notifier.telegramNotifier.isSendToThreadEnabled) {
      setNotifier({
        ...notifier,
        telegramNotifier: {
          ...notifier.telegramNotifier,
          isSendToThreadEnabled: true,
        },
      });
    }
  }, [notifier]);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.fields.botToken')}</div>
        <Input
          value={notifier?.telegramNotifier?.botToken || ''}
          onChange={(e) => {
            if (!notifier?.telegramNotifier) return;
            setNotifier({
              ...notifier,
              telegramNotifier: {
                ...notifier.telegramNotifier,
                botToken: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example token, not copy
          placeholder="1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        />
      </div>

      <div className="mb-1 sm:ml-[150px]">
        <a
          className="text-xs !text-blue-600"
          href="https://www.siteguarding.com/en/how-to-get-telegram-bot-api-token"
          target="_blank"
          rel="noreferrer"
        >
          {t('notifiers.telegram.botTokenHelpLink')}
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.fields.targetChatId')}
        </div>
        <div className="flex items-center">
          <Input
            value={notifier?.telegramNotifier?.targetChatId || ''}
            onChange={(e) => {
              if (!notifier?.telegramNotifier) return;

              setNotifier({
                ...notifier,
                telegramNotifier: {
                  ...notifier.telegramNotifier,
                  targetChatId: e.target.value.trim(),
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="-1001234567890"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.telegram.targetChatIdTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="max-w-[250px] sm:ml-[150px]">
        {!isShowHowToGetChatId ? (
          <div
            className="mt-1 cursor-pointer text-xs text-blue-600"
            onClick={() => setIsShowHowToGetChatId(true)}
          >
            {t('notifiers.telegram.chatIdHelp.toggle')}
          </div>
        ) : (
          <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            <Trans
              i18nKey="notifiers.telegram.chatIdHelp.privateChat"
              components={{ botLink: chatIdBotLink, underline: <u /> }}
            />
            <br />
            <br />
            <Trans
              i18nKey="notifiers.telegram.chatIdHelp.group"
              components={{ botLink: chatIdBotLink }}
            />
          </div>
        )}
      </div>

      <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.telegram.sendToGroupTopic')}
        </div>
        <div className="flex items-center">
          <Switch
            checked={notifier?.telegramNotifier?.isSendToThreadEnabled || false}
            onChange={(checked) => {
              if (!notifier?.telegramNotifier) return;

              setNotifier({
                ...notifier,
                telegramNotifier: {
                  ...notifier.telegramNotifier,
                  isSendToThreadEnabled: checked,
                  // Clear thread ID if disabling
                  threadId: checked ? notifier.telegramNotifier.threadId : undefined,
                },
              });
              setUnsaved();
            }}
            size="small"
          />

          <Tooltip
            className="cursor-pointer"
            title={t('notifiers.telegram.sendToGroupTopicTooltip')}
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      {notifier?.telegramNotifier?.isSendToThreadEnabled && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
              {t('notifiers.telegram.threadId')}
            </div>
            <div className="flex items-center">
              <Input
                value={notifier?.telegramNotifier?.threadId?.toString() || ''}
                onChange={(e) => {
                  if (!notifier?.telegramNotifier) return;

                  const value = e.target.value.trim();
                  const threadId = value ? parseInt(value, 10) : undefined;

                  setNotifier({
                    ...notifier,
                    telegramNotifier: {
                      ...notifier.telegramNotifier,
                      threadId: !isNaN(threadId!) ? threadId : undefined,
                    },
                  });
                  setUnsaved();
                }}
                size="small"
                className="w-full max-w-[250px]"
                placeholder="3"
                type="number"
                min="1"
              />

              <Tooltip className="cursor-pointer" title={t('notifiers.telegram.threadIdTooltip')}>
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="max-w-[250px] sm:ml-[150px]">
            <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {t('notifiers.telegram.threadIdHelp.steps')}
              <br />
              <br />
              <Trans
                i18nKey="notifiers.telegram.threadIdHelp.example"
                components={{
                  bold: <strong />,
                  code: <code className="rounded bg-gray-100 px-1" />,
                }}
              />
              <br />
              <br />
              <Trans
                i18nKey="notifiers.telegram.threadIdHelp.note"
                components={{ bold: <strong /> }}
              />
            </div>
          </div>
        </>
      )}

      <AdvancedSettingsToggleComponent
        label={t('notifiers.advancedSettings')}
        isShowAdvanced={isShowAdvanced}
        onToggle={() => setIsShowAdvanced(!isShowAdvanced)}
      />

      {isShowAdvanced && (
        <>
          <AdvancedFieldRowComponent
            label={t('notifiers.telegram.botApiUrl')}
            tooltip={t('notifiers.telegram.botApiUrlTooltip')}
          >
            <Input
              value={notifier?.telegramNotifier?.botApiUrl || ''}
              onChange={(e) => {
                if (!notifier?.telegramNotifier) return;

                setNotifier({
                  ...notifier,
                  telegramNotifier: {
                    ...notifier.telegramNotifier,
                    botApiUrl: e.target.value.trim(),
                  },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              // eslint-disable-next-line i18next/no-literal-string -- example URL, not copy
              placeholder="https://api.telegram.org"
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            label={t('notifiers.telegram.useProxy')}
            tooltip={t('notifiers.telegram.useProxyTooltip')}
          >
            <Switch
              checked={notifier?.telegramNotifier?.isProxyEnabled || false}
              onChange={(checked) => {
                if (!notifier?.telegramNotifier) return;

                setNotifier({
                  ...notifier,
                  telegramNotifier: {
                    ...notifier.telegramNotifier,
                    isProxyEnabled: checked,
                    proxyUrl: checked ? notifier.telegramNotifier.proxyUrl : undefined,
                  },
                });
                setUnsaved();
              }}
              size="small"
            />
          </AdvancedFieldRowComponent>

          {notifier?.telegramNotifier?.isProxyEnabled && (
            <AdvancedFieldRowComponent
              label={t('notifiers.telegram.proxyUrl')}
              tooltip={t('notifiers.telegram.proxyUrlTooltip')}
            >
              <Input
                value={notifier?.telegramNotifier?.proxyUrl || ''}
                onChange={(e) => {
                  if (!notifier?.telegramNotifier) return;

                  setNotifier({
                    ...notifier,
                    telegramNotifier: {
                      ...notifier.telegramNotifier,
                      proxyUrl: e.target.value.trim(),
                    },
                  });
                  setUnsaved();
                }}
                size="small"
                className="w-full max-w-[250px]"
                // eslint-disable-next-line i18next/no-literal-string -- example proxy URL
                placeholder="socks5://user:pass@host:1080"
              />
            </AdvancedFieldRowComponent>
          )}
        </>
      )}
    </>
  );
}

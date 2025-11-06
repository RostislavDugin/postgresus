import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditTelegramNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  const { t } = useTranslation(['notifier']);
  const [isShowHowToGetChatId, setIsShowHowToGetChatId] = useState(false);

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
      <div className="flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.bot_token_label')}</div>

        <div className="w-[250px]">
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
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            placeholder={t('notifier:form.bot_token_placeholder')}
          />
        </div>
      </div>

      <div className="mb-1 ml-[130px]">
        <a
          className="text-xs !text-blue-600"
          href="https://www.siteguarding.com/en/how-to-get-telegram-bot-api-token"
          target="_blank"
          rel="noreferrer"
        >
          {t('notifier:form.bot_token_link')}
        </a>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.target_chat_id_label')}</div>

        <div className="w-[250px]">
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
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            placeholder={t('notifier:form.target_chat_id_placeholder')}
          />
        </div>

        <Tooltip
          className="cursor-pointer"
          title={t('notifier:form.target_chat_id_tooltip')}
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      <div className="ml-[130px] max-w-[250px]">
        {!isShowHowToGetChatId ? (
          <div
            className="mt-1 cursor-pointer text-xs text-blue-600"
            onClick={() => setIsShowHowToGetChatId(true)}
          >
            {t('notifier:form.chat_id_link')}
          </div>
        ) : (
          <div className="mt-1 text-xs text-gray-500">
            {t('notifier:form.chat_id_help_1')}{' '}
            <a href="https://t.me/getmyid_bot" target="_blank" rel="noreferrer">
              @getmyid_bot
            </a>{' '}
            {t('notifier:form.chat_id_help_2')} <u>{t('notifier:form.chat_id_help_3')}</u>
            <br />
            <br />
            {t('notifier:form.chat_id_help_4')}{' '}
            <a href="https://t.me/getmyid_bot" target="_blank" rel="noreferrer">
              @getmyid_bot
            </a>{' '}
            {t('notifier:form.chat_id_help_5')}
          </div>
        )}
      </div>

      <div className="mt-4 mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px] break-all">{t('notifier:form.send_to_thread_label')}</div>

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
            setIsUnsaved(true);
          }}
          size="small"
        />

        <Tooltip
          className="cursor-pointer"
          title={t('notifier:form.send_to_thread_tooltip')}
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      {notifier?.telegramNotifier?.isSendToThreadEnabled && (
        <>
          <div className="mb-1 flex items-center">
            <div className="w-[130px] min-w-[130px]">{t('notifier:form.thread_id_label')}</div>

            <div className="w-[250px]">
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
                  setIsUnsaved(true);
                }}
                size="small"
                className="w-full"
                placeholder={t('notifier:form.thread_id_placeholder')}
                type="number"
                min="1"
              />
            </div>

            <Tooltip
              className="cursor-pointer"
              title={t('notifier:form.thread_id_tooltip')}
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>

          <div className="ml-[130px] max-w-[250px]">
            <div className="mt-1 text-xs text-gray-500">
              {t('notifier:form.thread_id_help_1')}
              <br />
              <br />
              <strong>{t('notifier:form.thread_id_help_example')}</strong> {t('notifier:form.thread_id_help_example_text')}{' '}
              <code className="rounded bg-gray-100 px-1">https://t.me/c/2831948048/3</code>, {t('notifier:form.thread_id_help_example_result')}{' '}
              <code className="rounded bg-gray-100 px-1">3</code>
              <br />
              <br />
              <strong>{t('notifier:form.thread_id_help_note')}</strong> {t('notifier:form.thread_id_help_note_text')}
            </div>
          </div>
        </>
      )}
    </>
  );
}

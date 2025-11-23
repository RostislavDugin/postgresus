import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditTelegramNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
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
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Bot token</div>
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
          How to get Telegram bot API token?
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Target chat ID</div>
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

          <Tooltip
            className="cursor-pointer"
            title="The chat where you want to receive the message (it can be your private chat or a group)"
          >
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
            How to get Telegram chat ID?
          </div>
        ) : (
          <div className="mt-1 text-xs text-gray-500">
            To get your chat ID, message{' '}
            <a href="https://t.me/getmyid_bot" target="_blank" rel="noreferrer">
              @getmyid_bot
            </a>{' '}
            in Telegram. <u>Make sure you started chat with the bot</u>
            <br />
            <br />
            If you want to get chat ID of a group, add your bot with{' '}
            <a href="https://t.me/getmyid_bot" target="_blank" rel="noreferrer">
              @getmyid_bot
            </a>{' '}
            to the group and write /start (you will see chat ID)
          </div>
        )}
      </div>

      <div className="mt-4 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Send to group topic</div>
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
            title="Enable this to send messages to a specific thread in a group chat"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      {notifier?.telegramNotifier?.isSendToThreadEnabled && (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Thread ID</div>
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

              <Tooltip
                className="cursor-pointer"
                title="The ID of the thread where messages should be sent"
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="max-w-[250px] sm:ml-[150px]">
            <div className="mt-1 text-xs text-gray-500">
              To get the thread ID, go to the thread in your Telegram group, tap on the thread name
              at the top, then tap &ldquo;Thread Info&rdquo;. Copy the thread link and take the last
              number from the URL.
              <br />
              <br />
              <strong>Example:</strong> If the thread link is{' '}
              <code className="rounded bg-gray-100 px-1">https://t.me/c/2831948048/3</code>, the
              thread ID is <code className="rounded bg-gray-100 px-1">3</code>
              <br />
              <br />
              <strong>Note:</strong> Thread functionality only works in group chats, not in private
              chats.
            </div>
          </div>
        </>
      )}
    </>
  );
}

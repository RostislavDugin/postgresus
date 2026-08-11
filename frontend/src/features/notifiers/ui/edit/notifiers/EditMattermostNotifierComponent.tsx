import { InfoCircleOutlined } from '@ant-design/icons';
import { Checkbox, Input, Select, Tooltip } from 'antd';
import { useState } from 'react';

import {
  MattermostDeliveryMode,
  type MattermostNotifier,
  type Notifier,
} from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditMattermostNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  const mattermostNotifier = notifier.mattermostNotifier;

  const [isShowOptional, setIsShowOptional] = useState(
    () =>
      !!(
        mattermostNotifier?.targetChannelName ||
        mattermostNotifier?.overrideUsername ||
        mattermostNotifier?.overrideIconUrl ||
        mattermostNotifier?.isInsecureSkipVerify
      ),
  );

  if (!mattermostNotifier) return <div />;

  const updateMattermostNotifier = (changes: Partial<MattermostNotifier>) => {
    setNotifier({
      ...notifier,
      mattermostNotifier: { ...mattermostNotifier, ...changes },
    });
    setUnsaved();
  };

  const isWebhookMode = mattermostNotifier.deliveryMode === MattermostDeliveryMode.WEBHOOK;

  return (
    <>
      <div className="mb-1 max-w-[250px] sm:ml-[150px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-blue-600"
          href="https://databasus.com/notifiers/mattermost"
          target="_blank"
          rel="noreferrer"
        >
          How to connect Mattermost?
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Connect via</div>
        <Select
          value={mattermostNotifier.deliveryMode}
          options={[
            { label: 'Incoming webhook', value: MattermostDeliveryMode.WEBHOOK },
            { label: 'Bot account', value: MattermostDeliveryMode.BOT },
          ]}
          onChange={(deliveryMode) => updateMattermostNotifier({ deliveryMode })}
          size="small"
          className="w-full max-w-[250px]"
        />
      </div>

      {isWebhookMode ? (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Incoming webhook URL</div>
          <Input
            value={mattermostNotifier.webhookUrl}
            onChange={(e) => updateMattermostNotifier({ webhookUrl: e.target.value.trim() })}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="https://mattermost.example.com/hooks/xxxxxxxx"
          />
        </div>
      ) : (
        <>
          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Server URL</div>
            <Input
              value={mattermostNotifier.serverUrl}
              onChange={(e) => updateMattermostNotifier({ serverUrl: e.target.value.trim() })}
              size="small"
              className="w-full max-w-[250px]"
              placeholder="https://mattermost.example.com"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Bot token</div>
            <Input
              value={mattermostNotifier.botToken}
              onChange={(e) => updateMattermostNotifier({ botToken: e.target.value.trim() })}
              size="small"
              className="w-full max-w-[250px]"
              placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxx"
            />
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Channel ID</div>
            <div className="flex items-center">
              <Input
                value={mattermostNotifier.targetChannelId}
                onChange={(e) =>
                  updateMattermostNotifier({ targetChannelId: e.target.value.trim() })
                }
                size="small"
                className="w-full max-w-[250px]"
                placeholder="8f4ycxjmztbwmcy1o3xasjrxha"
              />

              <Tooltip
                className="cursor-pointer"
                title="26-character channel ID, not the channel name. Open the channel, click its name and choose View Info."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mb-1 max-w-[250px] sm:ml-[150px]">
        <button
          type="button"
          onClick={() => setIsShowOptional(!isShowOptional)}
          className="text-xs text-blue-600 hover:underline"
        >
          {isShowOptional ? 'Hide optional settings' : 'Show optional settings'}
        </button>
      </div>

      {isShowOptional && (
        <>
          {isWebhookMode && (
            <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
              <div className="mb-1 min-w-[150px] sm:mb-0">Channel override</div>
              <div className="flex items-center">
                <Input
                  value={mattermostNotifier.targetChannelName}
                  onChange={(e) =>
                    updateMattermostNotifier({ targetChannelName: e.target.value.trim() })
                  }
                  size="small"
                  className="w-full max-w-[250px]"
                  placeholder="town-square"
                />

                <Tooltip
                  className="cursor-pointer"
                  title="Post to this channel instead of the one the webhook was created for. Mattermost ignores it when the webhook is locked to a channel."
                >
                  <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
                </Tooltip>
              </div>
            </div>
          )}

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Post as username</div>
            <div className="flex items-center">
              <Input
                value={mattermostNotifier.overrideUsername}
                onChange={(e) =>
                  updateMattermostNotifier({ overrideUsername: e.target.value.trim() })
                }
                size="small"
                className="w-full max-w-[250px]"
                placeholder="Databasus"
              />

              <Tooltip
                className="cursor-pointer"
                title="Requires Enable integrations to override usernames in the Mattermost system console, otherwise it is ignored."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Post as icon URL</div>
            <div className="flex items-center">
              <Input
                value={mattermostNotifier.overrideIconUrl}
                onChange={(e) =>
                  updateMattermostNotifier({ overrideIconUrl: e.target.value.trim() })
                }
                size="small"
                className="w-full max-w-[250px]"
                placeholder="https://databasus.com/icon.png"
              />

              <Tooltip
                className="cursor-pointer"
                title="Requires Enable integrations to override profile picture icons in the Mattermost system console, otherwise it is ignored."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>

          <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
            <div className="mb-1 min-w-[150px] sm:mb-0">Skip TLS verify</div>
            <div className="flex items-center">
              <Checkbox
                checked={mattermostNotifier.isInsecureSkipVerify}
                onChange={(e) =>
                  updateMattermostNotifier({ isInsecureSkipVerify: e.target.checked })
                }
              >
                Skip TLS
              </Checkbox>

              <Tooltip
                className="cursor-pointer"
                title="Skip TLS certificate verification. Enable this if your Mattermost server uses a self-signed certificate. Warning: this reduces security."
              >
                <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
              </Tooltip>
            </div>
          </div>
        </>
      )}

      <div className="mt-1 max-w-[250px] text-xs text-gray-500 sm:ml-[150px] dark:text-gray-400">
        {isWebhookMode ? (
          <>
            <strong>How to get an incoming webhook URL:</strong>
            <br />
            <br />
            1. Main menu - Integrations - Incoming Webhooks
            <br />
            2. Add Incoming Webhook, pick the channel
            <br />
            3. Copy the generated URL
          </>
        ) : (
          <>
            <strong>How to get a bot token:</strong>
            <br />
            <br />
            1. Integrations - Bot Accounts - Add Bot Account
            <br />
            2. Copy the token shown once after creation
            <br />
            3. Add the bot to the team and to the channel
          </>
        )}
      </div>
    </>
  );
}

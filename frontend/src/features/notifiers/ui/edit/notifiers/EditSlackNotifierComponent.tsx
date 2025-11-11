import { Input } from 'antd';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditSlackNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  return (
    <>
      <div className="mb-1 ml-[130px] max-w-[200px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-blue-600"
          href="https://postgresus.com/notifiers/slack"
          target="_blank"
          rel="noreferrer"
        >
          How to connect Slack (how to get bot token and chat ID)?
        </a>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">Bot token</div>

        <div className="w-[250px]">
          <Input
            value={notifier?.slackNotifier?.botToken || ''}
            onChange={(e) => {
              if (!notifier?.slackNotifier) return;

              setNotifier({
                ...notifier,
                slackNotifier: {
                  ...notifier.slackNotifier,
                  botToken: e.target.value.trim(),
                },
              });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            placeholder="xoxb-..."
          />
        </div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">Target chat ID</div>

        <div className="w-[250px]">
          <Input
            value={notifier?.slackNotifier?.targetChatId || ''}
            onChange={(e) => {
              if (!notifier?.slackNotifier) return;

              setNotifier({
                ...notifier,
                slackNotifier: {
                  ...notifier.slackNotifier,
                  targetChatId: e.target.value.trim(),
                },
              });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            placeholder="C1234567890"
          />
        </div>
      </div>
    </>
  );
}

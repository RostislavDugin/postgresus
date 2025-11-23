import { Input } from 'antd';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditDiscordNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Channel webhook URL</div>
        <Input
          value={notifier?.discordNotifier?.channelWebhookUrl || ''}
          onChange={(e) => {
            if (!notifier?.discordNotifier) return;
            setNotifier({
              ...notifier,
              discordNotifier: {
                ...notifier.discordNotifier,
                channelWebhookUrl: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        />
      </div>

      <div className="max-w-[250px] sm:ml-[150px]">
        <div className="mt-1 text-xs text-gray-500">
          <strong>How to get Discord webhook URL:</strong>
          <br />
          <br />
          1. Create or select a Discord channel
          <br />
          2. Go to channel settings (gear icon)
          <br />
          3. Navigate to Integrations
          <br />
          4. Create a new webhook
          <br />
          5. Copy the webhook URL
          <br />
          <br />
          <em>Note: make sure make channel private if needed</em>
        </div>
      </div>
    </>
  );
}

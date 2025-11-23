import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Select, Tooltip } from 'antd';

import type { Notifier } from '../../../../../entity/notifiers';
import { WebhookMethod } from '../../../../../entity/notifiers/models/webhook/WebhookMethod';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditWebhookNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Webhook URL</div>
        <Input
          value={notifier?.webhookNotifier?.webhookUrl || ''}
          onChange={(e) => {
            setNotifier({
              ...notifier,
              webhookNotifier: {
                ...(notifier.webhookNotifier || { webhookMethod: WebhookMethod.POST }),
                webhookUrl: e.target.value.trim(),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="https://example.com/webhook"
        />
      </div>

      <div className="mt-1 mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Method</div>
        <div className="flex items-center">
          <Select
            value={notifier?.webhookNotifier?.webhookMethod || WebhookMethod.POST}
            onChange={(value) => {
              setNotifier({
                ...notifier,
                webhookNotifier: {
                  ...(notifier.webhookNotifier || { webhookUrl: '' }),
                  webhookMethod: value,
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            options={[
              { value: WebhookMethod.POST, label: 'POST' },
              { value: WebhookMethod.GET, label: 'GET' },
            ]}
          />

          <Tooltip
            className="cursor-pointer"
            title="The HTTP method that will be used to call the webhook"
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      {notifier?.webhookNotifier?.webhookUrl && (
        <div className="mt-3">
          <div className="mb-1">Example request</div>

          {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.GET && (
            <div className="rounded bg-gray-100 p-2 px-3 text-sm break-all">
              GET {notifier?.webhookNotifier?.webhookUrl}?heading=✅ Backup completed for
              database&message=Backup completed successfully in 2m 17s.\nCompressed backup size:
              1.7GB
            </div>
          )}

          {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST && (
            <div className="rounded bg-gray-100 p-2 px-3 font-mono text-sm break-all whitespace-pre-line">
              {`POST ${notifier?.webhookNotifier?.webhookUrl}
Content-Type: application/json

{
  "heading": "✅ Backup completed for database",
  "message": "Backup completed successfully in 2m 17s.\\nCompressed backup size: 1.7GB"
}
`}
            </div>
          )}
        </div>
      )}
    </>
  );
}

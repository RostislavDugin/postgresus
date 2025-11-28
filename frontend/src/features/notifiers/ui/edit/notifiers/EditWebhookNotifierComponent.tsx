import { DeleteOutlined, InfoCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Input, Select, Tooltip } from 'antd';
import { useMemo } from 'react';

import type { Notifier, WebhookHeader } from '../../../../../entity/notifiers';
import { WebhookMethod } from '../../../../../entity/notifiers/models/webhook/WebhookMethod';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

const DEFAULT_BODY_TEMPLATE = `{
  "heading": "{{heading}}",
  "message": "{{message}}"
}`;

function validateJsonTemplate(template: string): string | null {
  if (!template.trim()) {
    return null; // Empty is valid (will use default)
  }

  // Replace placeholders with valid JSON strings before parsing
  const testJson = template.replace(/\{\{heading\}\}/g, 'test').replace(/\{\{message\}\}/g, 'test');

  try {
    JSON.parse(testJson);
    return null;
  } catch (e) {
    if (e instanceof SyntaxError) {
      return 'Invalid JSON format';
    }
    return 'Invalid JSON';
  }
}

export function EditWebhookNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  const headers = notifier?.webhookNotifier?.headers || [];
  const bodyTemplate = notifier?.webhookNotifier?.bodyTemplate || '';

  const jsonError = useMemo(() => validateJsonTemplate(bodyTemplate), [bodyTemplate]);

  const updateWebhookNotifier = (updates: Partial<typeof notifier.webhookNotifier>) => {
    setNotifier({
      ...notifier,
      webhookNotifier: {
        ...(notifier.webhookNotifier || { webhookUrl: '', webhookMethod: WebhookMethod.POST }),
        ...updates,
      },
    });
    setUnsaved();
  };

  const addHeader = () => {
    updateWebhookNotifier({
      headers: [...headers, { key: '', value: '' }],
    });
  };

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const newHeaders = [...headers];
    newHeaders[index] = { ...newHeaders[index], [field]: value };
    updateWebhookNotifier({ headers: newHeaders });
  };

  const removeHeader = (index: number) => {
    const newHeaders = headers.filter((_, i) => i !== index);
    updateWebhookNotifier({ headers: newHeaders });
  };

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Webhook URL</div>
        <Input
          value={notifier?.webhookNotifier?.webhookUrl || ''}
          onChange={(e) => {
            updateWebhookNotifier({ webhookUrl: e.target.value.trim() });
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
              updateWebhookNotifier({ webhookMethod: value });
            }}
            size="small"
            className="w-[100px] max-w-[250px]"
            options={[
              { value: WebhookMethod.POST, label: 'POST' },
              { value: WebhookMethod.GET, label: 'GET' },
            ]}
          />
        </div>
      </div>

      <div className="mt-3 mb-1 flex w-full flex-col items-start">
        <div className="mb-1 flex items-center">
          <span className="min-w-[150px]">
            Custom headers{' '}
            <Tooltip title="Add custom HTTP headers to the webhook request (e.g., Authorization, X-API-Key)">
              <InfoCircleOutlined className="ml-1" style={{ color: 'gray' }} />
            </Tooltip>
          </span>
        </div>

        <div className="w-full max-w-[500px]">
          {headers.map((header: WebhookHeader, index: number) => (
            <div key={index} className="mb-1 flex items-center gap-2">
              <Input
                value={header.key}
                onChange={(e) => updateHeader(index, 'key', e.target.value)}
                size="small"
                style={{ width: 150, flexShrink: 0 }}
                placeholder="Header name"
              />
              <Input
                value={header.value}
                onChange={(e) => updateHeader(index, 'value', e.target.value)}
                size="small"
                style={{ flex: 1, minWidth: 0 }}
                placeholder="Header value"
              />
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                onClick={() => removeHeader(index)}
              />
            </div>
          ))}

          <Button
            type="dashed"
            size="small"
            icon={<PlusOutlined />}
            onClick={addHeader}
            className="mt-1"
          >
            Add header
          </Button>
        </div>
      </div>

      {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST && (
        <div className="mt-3 mb-1 flex w-full flex-col items-start">
          <div className="mb-1 flex items-center">
            <span className="min-w-[150px]">Body template </span>
          </div>

          <div className="mb-2 text-xs text-gray-500 dark:text-gray-400">
            <span className="mr-4">
              <code className="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700">
                {'{{heading}}'}
              </code>{' '}
              — notification title
            </span>
            <span>
              <code className="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700">
                {'{{message}}'}
              </code>{' '}
              — notification message
            </span>
          </div>

          <Input.TextArea
            value={bodyTemplate}
            onChange={(e) => {
              updateWebhookNotifier({ bodyTemplate: e.target.value });
            }}
            className="w-full max-w-[500px] font-mono text-xs"
            rows={6}
            placeholder={DEFAULT_BODY_TEMPLATE}
            status={jsonError ? 'error' : undefined}
          />
          {jsonError && <div className="mt-1 text-xs text-red-500">{jsonError}</div>}
        </div>
      )}

      {notifier?.webhookNotifier?.webhookUrl && (
        <div className="mt-4">
          <div className="mb-1 font-medium">Example request</div>

          {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.GET && (
            <div className="rounded bg-gray-100 p-2 px-3 text-sm break-all dark:bg-gray-800">
              <div className="font-semibold text-blue-600 dark:text-blue-400">GET</div>
              <div className="mt-1">
                {notifier?.webhookNotifier?.webhookUrl}
                {
                  '?heading=✅ Backup completed for database "my-database" (workspace "Production")&message=Backup completed successfully in 1m 23s.%0ACompressed backup size: 256.00 MB'
                }
              </div>
              {headers.length > 0 && (
                <div className="mt-2 border-t border-gray-200 pt-2 dark:border-gray-600">
                  <div className="text-xs font-semibold text-gray-500 dark:text-gray-400">
                    Headers:
                  </div>
                  {headers
                    .filter((h) => h.key)
                    .map((h, i) => (
                      <div key={i} className="text-xs">
                        {h.key}: {h.value || '(empty)'}
                      </div>
                    ))}
                </div>
              )}
            </div>
          )}

          {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST && (
            <div className="rounded bg-gray-100 p-2 px-3 font-mono text-sm break-all whitespace-pre-line dark:bg-gray-800">
              <div className="font-semibold text-blue-600 dark:text-blue-400">
                POST {notifier?.webhookNotifier?.webhookUrl}
              </div>
              <div className="mt-1 text-gray-600 dark:text-gray-400">
                {headers.find((h) => h.key.toLowerCase() === 'content-type')
                  ? ''
                  : 'Content-Type: application/json'}
                {headers
                  .filter((h) => h.key)
                  .map((h) => `\n${h.key}: ${h.value}`)
                  .join('')}
              </div>
              <div className="mt-2 whitespace-pre">
                {notifier?.webhookNotifier?.bodyTemplate
                  ? notifier.webhookNotifier.bodyTemplate
                      .replace(
                        '{{heading}}',
                        '✅ Backup completed for database "my-database" (workspace "Production")',
                      )
                      .replace(
                        '{{message}}',
                        'Backup completed successfully in 1m 23s.\\nCompressed backup size: 256.00 MB',
                      )
                  : `{
  "heading": "✅ Backup completed for database \\"my-database\\" (workspace \\"My workspace\\")",
  "message": "Backup completed successfully in 1m 23s.\\nCompressed backup size: 256.00 MB"
}`}
              </div>
            </div>
          )}
        </div>
      )}
    </>
  );
}

import type { Notifier, WebhookHeader } from '../../../../../entity/notifiers';
import { WebhookMethod } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowWebhookNotifierComponent({ notifier }: Props) {
  const headers = notifier?.webhookNotifier?.headers || [];
  const hasHeaders = headers.filter((h: WebhookHeader) => h.key).length > 0;

  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">Webhook URL</div>
        <div className="max-w-[350px] truncate">{notifier?.webhookNotifier?.webhookUrl || '-'}</div>
      </div>

      <div className="mt-1 mb-1 flex items-center">
        <div className="min-w-[110px]">Method</div>
        <div>{notifier?.webhookNotifier?.webhookMethod || '-'}</div>
      </div>

      {hasHeaders && (
        <div className="mt-1 mb-1 flex items-start">
          <div className="min-w-[110px]">Headers</div>
          <div className="flex flex-col text-sm">
            {headers
              .filter((h: WebhookHeader) => h.key)
              .map((h: WebhookHeader, i: number) => (
                <div key={i} className="text-gray-600">
                  <span className="font-medium">{h.key}:</span> {h.value || '(empty)'}
                </div>
              ))}
          </div>
        </div>
      )}

      {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST &&
        notifier?.webhookNotifier?.bodyTemplate && (
          <div className="mt-1 mb-1 flex items-start">
            <div className="min-w-[110px]">Body Template</div>
            <div className="max-w-[350px] rounded bg-gray-50 p-2 font-mono text-xs whitespace-pre-wrap">
              {notifier.webhookNotifier.bodyTemplate}
            </div>
          </div>
        )}
    </>
  );
}

import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowWebhookNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation(['notifier', 'common']);
  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.webhook_url_label')}</div>

        <div className="w-[250px]">{notifier?.webhookNotifier?.webhookUrl || '-'}</div>
      </div>

      <div className="mt-1 mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.webhook_method_label')}</div>
        <div>{notifier?.webhookNotifier?.webhookMethod || '-'}</div>
      </div>

      {notifier?.webhookNotifier?.customTemplate && (
        <div className="mt-1 mb-1">
          <div className="min-w-[110px] mb-1 font-medium">{t('notifier:form.custom_template_label')}</div>
          <div className="rounded bg-gray-100 p-2 font-mono text-xs break-all whitespace-pre-wrap">
            {notifier.webhookNotifier.customTemplate}
          </div>
        </div>
      )}
    </>
  );
}

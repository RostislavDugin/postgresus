import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowTelegramNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.fields.botToken')}</div>

        <div>*********</div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px] pr-2">{t('notifiers.fields.targetChatId')}</div>
        {notifier?.telegramNotifier?.targetChatId}
      </div>

      {notifier?.telegramNotifier?.threadId && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('notifiers.telegram.topicId')}</div>
          {notifier.telegramNotifier.threadId}
        </div>
      )}

      {notifier?.telegramNotifier?.botApiUrl && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('notifiers.telegram.botApiUrl')}</div>
          {notifier.telegramNotifier.botApiUrl}
        </div>
      )}

      {notifier?.telegramNotifier?.isProxyEnabled && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px] pr-2">{t('notifiers.telegram.proxy')}</div>
          {t('common.states.enabled')}
        </div>
      )}
    </>
  );
}

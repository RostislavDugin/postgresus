import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowTelegramNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation(['notifier']);
  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.bot_token_label')}</div>

        <div className="w-[250px]">*********</div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.target_chat_id_label')}</div>
        {notifier?.telegramNotifier?.targetChatId}
      </div>

      {notifier?.telegramNotifier?.threadId && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">{t('notifier:form.thread_id_label')}</div>
          {notifier.telegramNotifier.threadId}
        </div>
      )}
    </>
  );
}

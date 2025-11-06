import { Input } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditSlackNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  const { t } = useTranslation(['notifier']);

  return (
    <>
      <div className="mb-1 ml-[130px] max-w-[200px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-blue-600"
          href="https://postgresus.com/notifier-slack"
          target="_blank"
          rel="noreferrer"
        >
          {t('notifier:form.slack_link')}
        </a>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.bot_token_label')}</div>

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
            placeholder={t('notifier:form.slack_bot_token_placeholder')}
          />
        </div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.target_chat_id_label')}</div>

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
            placeholder={t('notifier:form.slack_chat_id_placeholder')}
          />
        </div>
      </div>
    </>
  );
}

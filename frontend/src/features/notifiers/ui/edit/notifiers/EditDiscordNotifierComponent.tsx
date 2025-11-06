import { Input } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditDiscordNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  const { t } = useTranslation(['notifier']);

  return (
    <>
      <div className="flex">
        <div className="w-[130px] max-w-[130px] min-w-[130px] pr-3">{t('notifier:form.discord_webhook_url_label')}</div>

        <div className="w-[250px]">
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
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            placeholder={t('notifier:form.discord_webhook_url_placeholder')}
          />
        </div>
      </div>

      <div className="ml-[130px] max-w-[250px]">
        <div className="mt-1 text-xs text-gray-500">
          <strong>{t('notifier:form.discord_help_title')}</strong>
          <br />
          <br />
          {t('notifier:form.discord_help_step1')}
          <br />
          {t('notifier:form.discord_help_step2')}
          <br />
          {t('notifier:form.discord_help_step3')}
          <br />
          {t('notifier:form.discord_help_step4')}
          <br />
          {t('notifier:form.discord_help_step5')}
          <br />
          <br />
          <em>{t('notifier:form.discord_help_note')}</em>
        </div>
      </div>
    </>
  );
}

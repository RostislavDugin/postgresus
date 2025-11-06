import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowEmailNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation(['notifier']);

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.target_email_label')}</div>
        {notifier?.emailNotifier?.targetEmail}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.smtp_host_label')}</div>
        {notifier?.emailNotifier?.smtpHost}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.smtp_port_label')}</div>
        {notifier?.emailNotifier?.smtpPort}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.smtp_user_label')}</div>
        {notifier?.emailNotifier?.smtpUser}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.smtp_password_label')}</div>
        {notifier?.emailNotifier?.smtpPassword ? '*********' : ''}
      </div>
    </>
  );
}

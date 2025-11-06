import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditEmailNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  const { t } = useTranslation(['notifier']);

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.target_email_label')}</div>
        <Input
          value={notifier?.emailNotifier?.targetEmail || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                targetEmail: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('notifier:form.target_email_placeholder')}
        />

        <Tooltip className="cursor-pointer" title={t('notifier:form.target_email_tooltip')}>
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.smtp_host_label')}</div>
        <Input
          value={notifier?.emailNotifier?.smtpHost || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpHost: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('notifier:form.smtp_host_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.smtp_port_label')}</div>
        <Input
          type="number"
          value={notifier?.emailNotifier?.smtpPort || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpPort: Number(e.target.value),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('notifier:form.smtp_port_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.smtp_user_label')}</div>
        <Input
          value={notifier?.emailNotifier?.smtpUser || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpUser: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('notifier:form.smtp_user_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.smtp_password_label')}</div>
        <Input
          value={notifier?.emailNotifier?.smtpPassword || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpPassword: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('notifier:form.smtp_password_placeholder')}
        />
      </div>
    </>
  );
}

import { InfoCircleOutlined } from '@ant-design/icons';
import { Checkbox, Input, Select, Tooltip } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  EMAIL_NOTIFIER_SECURITY_LABEL_KEYS,
  EmailNotifierSecurity,
  type Notifier,
  getDefaultEmailNotifierSecurity,
  getSecurityAfterPortChange,
} from '../../../../../entity/notifiers';
import {
  AdvancedFieldRowComponent,
  AdvancedSettingsToggleComponent,
} from '../../../../../shared/ui';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setUnsaved: () => void;
}

export function EditEmailNotifierComponent({ notifier, setNotifier, setUnsaved }: Props) {
  const { t } = useTranslation();
  const emailNotifier = notifier?.emailNotifier;
  const hasAdvancedValues =
    !!emailNotifier?.isInsecureSkipVerify ||
    !!emailNotifier?.heloName ||
    (!!emailNotifier &&
      emailNotifier.security !== getDefaultEmailNotifierSecurity(emailNotifier.smtpPort));
  const [showAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.targetEmail')}</div>
        <div className="flex items-center">
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
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="example@gmail.com"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.email.targetEmailTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.smtpHost')}</div>
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
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          // eslint-disable-next-line i18next/no-literal-string -- example hostname
          placeholder="smtp.gmail.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.smtpPort')}</div>
        <Input
          type="number"
          value={notifier?.emailNotifier?.smtpPort || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            const nextPort = Number(e.target.value);

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpPort: nextPort,
                security: getSecurityAfterPortChange(
                  notifier.emailNotifier.security,
                  notifier.emailNotifier.smtpPort,
                  nextPort,
                ),
              },
            });
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="25"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.smtpUser')}</div>
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
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="user@gmail.com"
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">
          {t('notifiers.email.smtpPassword')}
        </div>
        <Input
          type="password"
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
            setUnsaved();
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('notifiers.email.smtpPasswordPlaceholder')}
        />
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{t('notifiers.email.from')}</div>
        <div className="flex items-center">
          <Input
            value={notifier?.emailNotifier?.from || ''}
            onChange={(e) => {
              if (!notifier?.emailNotifier) return;

              setNotifier({
                ...notifier,
                emailNotifier: {
                  ...notifier.emailNotifier,
                  from: e.target.value.trim(),
                },
              });
              setUnsaved();
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="example@example.com"
          />

          <Tooltip className="cursor-pointer" title={t('notifiers.email.fromTooltip')}>
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <AdvancedSettingsToggleComponent
        label={t('notifiers.advancedSettings')}
        isShowAdvanced={showAdvanced}
        onToggle={() => setShowAdvanced(!showAdvanced)}
      />

      {showAdvanced && (
        <>
          <AdvancedFieldRowComponent
            label={t('notifiers.email.security')}
            tooltip={t('notifiers.email.securityTooltip')}
          >
            <Select
              value={notifier?.emailNotifier?.security}
              onChange={(security: EmailNotifierSecurity) => {
                if (!notifier?.emailNotifier) return;

                setNotifier({
                  ...notifier,
                  emailNotifier: { ...notifier.emailNotifier, security },
                });
                setUnsaved();
              }}
              size="small"
              className="w-[250px] max-w-[250px]"
              options={Object.values(EmailNotifierSecurity).map((security) => ({
                value: security,
                label: t(EMAIL_NOTIFIER_SECURITY_LABEL_KEYS[security]),
              }))}
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            label={t('notifiers.email.heloName')}
            tooltip={t('notifiers.email.heloNameTooltip')}
          >
            <Input
              value={notifier?.emailNotifier?.heloName || ''}
              onChange={(e) => {
                if (!notifier?.emailNotifier) return;

                setNotifier({
                  ...notifier,
                  emailNotifier: { ...notifier.emailNotifier, heloName: e.target.value.trim() },
                });
                setUnsaved();
              }}
              size="small"
              className="w-full max-w-[250px]"
              placeholder={t('notifiers.email.heloNamePlaceholder')}
            />
          </AdvancedFieldRowComponent>

          <AdvancedFieldRowComponent
            label={t('notifiers.fields.skipTlsVerify')}
            tooltip={t('notifiers.email.skipTlsTooltip')}
          >
            <Checkbox
              checked={notifier?.emailNotifier?.isInsecureSkipVerify || false}
              onChange={(e) => {
                if (!notifier?.emailNotifier) return;

                setNotifier({
                  ...notifier,
                  emailNotifier: {
                    ...notifier.emailNotifier,
                    isInsecureSkipVerify: e.target.checked,
                  },
                });
                setUnsaved();
              }}
            >
              {t('notifiers.fields.skipTls')}
            </Checkbox>
          </AdvancedFieldRowComponent>
        </>
      )}
    </>
  );
}

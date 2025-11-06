import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  type Notifier,
  NotifierType,
  WebhookMethod,
  notifierApi,
  validateDiscordNotifier,
  validateEmailNotifier,
  validateSlackNotifier,
  validateTeamsNotifier,
  validateTelegramNotifier,
  validateWebhookNotifier,
} from '../../../../entity/notifiers';
import { getNotifierLogoFromType } from '../../../../entity/notifiers/models/getNotifierLogoFromType';
import { ToastHelper } from '../../../../shared/toast';
import { EditDiscordNotifierComponent } from './notifiers/EditDiscordNotifierComponent';
import { EditEmailNotifierComponent } from './notifiers/EditEmailNotifierComponent';
import { EditSlackNotifierComponent } from './notifiers/EditSlackNotifierComponent';
import { EditTeamsNotifierComponent } from './notifiers/EditTeamsNotifierComponent';
import { EditTelegramNotifierComponent } from './notifiers/EditTelegramNotifierComponent';
import { EditWebhookNotifierComponent } from './notifiers/EditWebhookNotifierComponent';

interface Props {
  isShowClose: boolean;
  onClose: () => void;

  isShowName: boolean;

  editingNotifier?: Notifier;
  onChanged: (notifier: Notifier) => void;
}

export function EditNotifierComponent({
  isShowClose,
  onClose,
  isShowName,
  editingNotifier,
  onChanged,
}: Props) {
  const { t } = useTranslation(['notifier', 'common']);
  const [notifier, setNotifier] = useState<Notifier | undefined>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [isSendingTestNotification, setIsSendingTestNotification] = useState(false);
  const [isTestNotificationSuccess, setIsTestNotificationSuccess] = useState(false);

  const save = async () => {
    if (!notifier) return;

    setIsSaving(true);

    try {
      await notifierApi.saveNotifier(notifier);
      onChanged(notifier);
      setIsUnsaved(false);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsSaving(false);
  };

  const sendTestNotification = async () => {
    if (!notifier) return;

    setIsSendingTestNotification(true);

    try {
      await notifierApi.sendTestNotificationDirect(notifier);
      setIsTestNotificationSuccess(true);
      ToastHelper.showToast({
        title: t('notifier:message.test_notification_sent_title'),
        description: t('notifier:message.test_notification_sent_description'),
      });
    } catch (e) {
      alert((e as Error).message);

      if (notifier.notifierType === NotifierType.SLACK) {
        alert(
          t('notifier:message.slack_error_hint'),
        );
      }
    }

    setIsSendingTestNotification(false);
  };

  const setNotifierType = (type: NotifierType) => {
    if (!notifier) return;

    notifier.emailNotifier = undefined;
    notifier.telegramNotifier = undefined;
    notifier.teamsNotifier = undefined;

    if (type === NotifierType.TELEGRAM) {
      notifier.telegramNotifier = {
        botToken: '',
        targetChatId: '',
      };
    }

    if (type === NotifierType.EMAIL) {
      notifier.emailNotifier = {
        targetEmail: '',
        smtpHost: '',
        smtpPort: 0,
        smtpUser: '',
        smtpPassword: '',
      };
    }

    if (type === NotifierType.WEBHOOK) {
      notifier.webhookNotifier = {
        webhookUrl: '',
        webhookMethod: WebhookMethod.POST,
      };
    }

    if (type === NotifierType.SLACK) {
      notifier.slackNotifier = {
        botToken: '',
        targetChatId: '',
      };
    }

    if (type === NotifierType.DISCORD) {
      notifier.discordNotifier = {
        channelWebhookUrl: '',
      };
    }

    if (type === NotifierType.TEAMS) {
      notifier.teamsNotifier = { powerAutomateUrl: '' };
    }

    setNotifier(
      JSON.parse(
        JSON.stringify({
          ...notifier,
          notifierType: type,
        }),
      ),
    );
  };

  useEffect(() => {
    setIsUnsaved(false);
    setNotifier(
      editingNotifier
        ? JSON.parse(JSON.stringify(editingNotifier))
        : {
            id: undefined as unknown as string,
            name: '',
            notifierType: NotifierType.TELEGRAM,
            telegramNotifier: {
              botToken: '',
              targetChatId: '',
            },
          },
    );
  }, [editingNotifier]);

  useEffect(() => {
    setIsTestNotificationSuccess(false);
  }, [notifier]);

  const isAllDataFilled = () => {
    if (!notifier) return false;
    if (!notifier.name) return false;

    if (notifier.notifierType === NotifierType.TELEGRAM && notifier.telegramNotifier) {
      return validateTelegramNotifier(notifier.telegramNotifier);
    }

    if (notifier.notifierType === NotifierType.EMAIL && notifier.emailNotifier) {
      return validateEmailNotifier(notifier.emailNotifier);
    }

    if (notifier.notifierType === NotifierType.WEBHOOK && notifier.webhookNotifier) {
      return validateWebhookNotifier(notifier.webhookNotifier);
    }

    if (notifier.notifierType === NotifierType.SLACK && notifier.slackNotifier) {
      return validateSlackNotifier(notifier.slackNotifier);
    }

    if (notifier.notifierType === NotifierType.DISCORD && notifier.discordNotifier) {
      return validateDiscordNotifier(notifier.discordNotifier);
    }

    if (notifier.notifierType === NotifierType.TEAMS && notifier.teamsNotifier) {
      return validateTeamsNotifier(notifier.teamsNotifier);
    }

    return false;
  };

  if (!notifier) return <div />;

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[130px]">{t('common:common.name')}</div>

          <Input
            value={notifier?.name || ''}
            onChange={(e) => {
              setNotifier({ ...notifier, name: e.target.value });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder={t('notifier:form.name_placeholder')}
          />
        </div>
      )}

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('common:common.type')}</div>

        <Select
          value={notifier?.notifierType}
          options={[
            { label: 'Telegram', value: NotifierType.TELEGRAM },
            { label: 'Email', value: NotifierType.EMAIL },
            { label: 'Webhook', value: NotifierType.WEBHOOK },
            { label: 'Slack', value: NotifierType.SLACK },
            { label: 'Discord', value: NotifierType.DISCORD },
            { label: 'Teams', value: NotifierType.TEAMS },
          ]}
          onChange={(value) => {
            setNotifierType(value);
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
        />

        <img src={getNotifierLogoFromType(notifier?.notifierType)} className="ml-2 h-4 w-4" />
      </div>

      <div className="mt-5" />

      <div>
        {notifier?.notifierType === NotifierType.TELEGRAM && (
          <EditTelegramNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setIsUnsaved={setIsUnsaved}
          />
        )}

        {notifier?.notifierType === NotifierType.EMAIL && (
          <EditEmailNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setIsUnsaved={setIsUnsaved}
          />
        )}

        {notifier?.notifierType === NotifierType.WEBHOOK && (
          <EditWebhookNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setIsUnsaved={setIsUnsaved}
          />
        )}

        {notifier?.notifierType === NotifierType.SLACK && (
          <EditSlackNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setIsUnsaved={setIsUnsaved}
          />
        )}

        {notifier?.notifierType === NotifierType.DISCORD && (
          <EditDiscordNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setIsUnsaved={setIsUnsaved}
          />
        )}
        {notifier?.notifierType === NotifierType.TEAMS && (
          <EditTeamsNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setIsUnsaved={setIsUnsaved}
          />
        )}
      </div>

      <div className="mt-3 flex">
        {isUnsaved && !isTestNotificationSuccess ? (
          <Button
            className="mr-1"
            disabled={isSendingTestNotification || !isAllDataFilled()}
            loading={isSendingTestNotification}
            type="primary"
            onClick={sendTestNotification}
          >
            {t('notifier:action.send_test')}
          </Button>
        ) : (
          <div />
        )}

        {isUnsaved && isTestNotificationSuccess ? (
          <Button
            className="mr-1"
            disabled={isSaving || !isAllDataFilled()}
            loading={isSaving}
            type="primary"
            onClick={save}
          >
            {t('common:common.save')}
          </Button>
        ) : (
          <div />
        )}

        {isShowClose ? (
          <Button
            className="mr-1"
            disabled={isSaving}
            type="primary"
            danger
            ghost
            onClick={onClose}
          >
            {t('common:common.cancel')}
          </Button>
        ) : (
          <div />
        )}
      </div>
    </div>
  );
}

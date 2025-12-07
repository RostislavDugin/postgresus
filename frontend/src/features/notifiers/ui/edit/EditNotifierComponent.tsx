import { Button, Input, Select } from 'antd';
import { useEffect, useState } from 'react';

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
  workspaceId: string;

  isShowClose: boolean;
  onClose: () => void;

  isShowName: boolean;

  editingNotifier?: Notifier;
  onChanged: (notifier: Notifier) => void;
}

export function EditNotifierComponent({
  workspaceId,
  isShowClose,
  onClose,
  isShowName,
  editingNotifier,
  onChanged,
}: Props) {
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
        title: 'Test notification sent!',
        description: 'Test notification sent successfully',
      });
    } catch (e) {
      alert((e as Error).message);

      if (notifier.notifierType === NotifierType.SLACK) {
        alert(
          'Make sure channel is public or bot is added to the private channel (via @invite) or group. For direct messages use User ID from Slack profile.',
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
        from: '',
      };
    }

    if (type === NotifierType.WEBHOOK) {
      notifier.webhookNotifier = {
        webhookUrl: '',
        webhookMethod: WebhookMethod.POST,
        headers: [],
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
          workspaceId,
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
            workspaceId,
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
      return validateTelegramNotifier(!notifier.id, notifier.telegramNotifier);
    }

    if (notifier.notifierType === NotifierType.EMAIL && notifier.emailNotifier) {
      return validateEmailNotifier(notifier.emailNotifier);
    }

    if (notifier.notifierType === NotifierType.WEBHOOK && notifier.webhookNotifier) {
      return validateWebhookNotifier(!notifier.id, notifier.webhookNotifier);
    }

    if (notifier.notifierType === NotifierType.SLACK && notifier.slackNotifier) {
      return validateSlackNotifier(!notifier.id, notifier.slackNotifier);
    }

    if (notifier.notifierType === NotifierType.DISCORD && notifier.discordNotifier) {
      return validateDiscordNotifier(!notifier.id, notifier.discordNotifier);
    }

    if (notifier.notifierType === NotifierType.TEAMS && notifier.teamsNotifier) {
      return validateTeamsNotifier(!notifier.id, notifier.teamsNotifier);
    }

    return false;
  };

  if (!notifier) return <div />;

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
          <div className="mb-1 min-w-[150px] sm:mb-0">Name</div>

          <Input
            value={notifier?.name || ''}
            onChange={(e) => {
              setNotifier({ ...notifier, name: e.target.value });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full max-w-[250px]"
            placeholder="Chat with me"
          />
        </div>
      )}

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-[150px] sm:mb-0">Type</div>

        <div className="flex items-center">
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
            className="w-[250px] max-w-[250px]"
          />

          <img src={getNotifierLogoFromType(notifier?.notifierType)} className="ml-2 h-4 w-4" />
        </div>
      </div>

      <div className="mt-5" />

      <div>
        {notifier?.notifierType === NotifierType.TELEGRAM && (
          <EditTelegramNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.EMAIL && (
          <EditEmailNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.WEBHOOK && (
          <EditWebhookNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.SLACK && (
          <EditSlackNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}

        {notifier?.notifierType === NotifierType.DISCORD && (
          <EditDiscordNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
          />
        )}
        {notifier?.notifierType === NotifierType.TEAMS && (
          <EditTeamsNotifierComponent
            notifier={notifier}
            setNotifier={setNotifier}
            setUnsaved={() => {
              setIsUnsaved(true);
              setIsTestNotificationSuccess(false);
            }}
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
            Send test notification
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
            Save
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
            Cancel
          </Button>
        ) : (
          <div />
        )}
      </div>
    </div>
  );
}

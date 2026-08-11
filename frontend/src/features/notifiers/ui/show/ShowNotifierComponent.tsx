import { type Notifier, NotifierType } from '../../../../entity/notifiers';
import { getNotifierLogoFromType } from '../../../../entity/notifiers/models/getNotifierLogoFromType';
import { getNotifierNameFromType } from '../../../../entity/notifiers/models/getNotifierNameFromType';
import { ShowDiscordNotifierComponent } from './notifier/ShowDiscordNotifierComponent';
import { ShowEmailNotifierComponent } from './notifier/ShowEmailNotifierComponent';
import { ShowMattermostNotifierComponent } from './notifier/ShowMattermostNotifierComponent';
import { ShowSlackNotifierComponent } from './notifier/ShowSlackNotifierComponent';
import { ShowTeamsNotifierComponent } from './notifier/ShowTeamsNotifierComponent';
import { ShowTelegramNotifierComponent } from './notifier/ShowTelegramNotifierComponent';
import { ShowWebhookNotifierComponent } from './notifier/ShowWebhookNotifierComponent';

interface Props {
  notifier: Notifier;
}

export function ShowNotifierComponent({ notifier }: Props) {
  return (
    <div>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Type</div>

        {getNotifierNameFromType(notifier?.notifierType)}
        <img src={getNotifierLogoFromType(notifier?.notifierType)} className="ml-1 h-4 w-4" />
      </div>

      <div>
        {notifier?.notifierType === NotifierType.TELEGRAM && (
          <ShowTelegramNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.EMAIL && (
          <ShowEmailNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.WEBHOOK && (
          <ShowWebhookNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.SLACK && (
          <ShowSlackNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.DISCORD && (
          <ShowDiscordNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.TEAMS && (
          <ShowTeamsNotifierComponent notifier={notifier} />
        )}

        {notifier?.notifierType === NotifierType.MATTERMOST && (
          <ShowMattermostNotifierComponent notifier={notifier} />
        )}
      </div>
    </div>
  );
}

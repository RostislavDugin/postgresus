import { MattermostDeliveryMode, type Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowMattermostNotifierComponent({ notifier }: Props) {
  const mattermostNotifier = notifier.mattermostNotifier;

  if (!mattermostNotifier) return <div />;

  const isWebhookMode = mattermostNotifier.deliveryMode === MattermostDeliveryMode.WEBHOOK;

  return (
    <>
      <div className="flex">
        <div className="max-w-[110px] min-w-[110px] pr-3">Connect via</div>
        <div>{isWebhookMode ? 'Incoming webhook' : 'Bot account'}</div>
      </div>

      {isWebhookMode ? (
        <>
          <div className="flex">
            <div className="max-w-[110px] min-w-[110px] pr-3">Webhook URL</div>
            <div className="break-all">
              {mattermostNotifier.webhookUrl
                ? `${mattermostNotifier.webhookUrl}*******`
                : '*******'}
            </div>
          </div>

          {mattermostNotifier.targetChannelName && (
            <div className="flex">
              <div className="max-w-[110px] min-w-[110px] pr-3">Channel</div>
              <div className="break-all">{mattermostNotifier.targetChannelName}</div>
            </div>
          )}
        </>
      ) : (
        <>
          <div className="flex">
            <div className="max-w-[110px] min-w-[110px] pr-3">Server URL</div>
            <div className="break-all">{mattermostNotifier.serverUrl}</div>
          </div>

          <div className="flex">
            <div className="max-w-[110px] min-w-[110px] pr-3">Channel ID</div>
            <div className="break-all">{mattermostNotifier.targetChannelId}</div>
          </div>
        </>
      )}
    </>
  );
}

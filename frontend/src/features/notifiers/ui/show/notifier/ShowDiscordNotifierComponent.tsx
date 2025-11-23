import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowDiscordNotifierComponent({ notifier }: Props) {
  return (
    <>
      <div className="flex">
        <div className="max-w-[110px] min-w-[110px] pr-3">Channel webhook URL</div>

        <div>{notifier.webhookNotifier?.webhookUrl.slice(0, 10)}*******</div>
      </div>
    </>
  );
}

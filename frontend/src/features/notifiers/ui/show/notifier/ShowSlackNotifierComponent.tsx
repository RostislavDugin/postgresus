import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowSlackNotifierComponent({ notifier }: Props) {
  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">Bot token</div>

        <div>*********</div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Target chat ID</div>
        {notifier?.slackNotifier?.targetChatId}
      </div>
    </>
  );
}

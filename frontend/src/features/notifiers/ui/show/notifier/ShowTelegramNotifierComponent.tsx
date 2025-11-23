import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowTelegramNotifierComponent({ notifier }: Props) {
  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">Bot token</div>

        <div>*********</div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Target chat ID</div>
        {notifier?.telegramNotifier?.targetChatId}
      </div>

      {notifier?.telegramNotifier?.threadId && (
        <div className="mb-1 flex items-center">
          <div className="min-w-[110px]">Topic ID</div>
          {notifier.telegramNotifier.threadId}
        </div>
      )}
    </>
  );
}

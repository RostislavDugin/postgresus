import type { TelegramNotifier } from './TelegramNotifier';

export const validateTelegramNotifier = (
  isCreate: boolean,
  notifier: TelegramNotifier,
): boolean => {
  if (isCreate && !notifier.botToken) {
    return false;
  }

  if (!notifier.targetChatId) {
    return false;
  }

  // If thread is enabled, thread ID must be present and valid
  if (notifier.isSendToThreadEnabled && (!notifier.threadId || notifier.threadId <= 0)) {
    return false;
  }

  return true;
};

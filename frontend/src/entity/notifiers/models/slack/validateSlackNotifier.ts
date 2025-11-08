import type { SlackNotifier } from './SlackNotifier';

export const validateSlackNotifier = (isCreate: boolean, notifier: SlackNotifier): boolean => {
  if (isCreate && !notifier.botToken) {
    return false;
  }

  if (!notifier.targetChatId) {
    return false;
  }

  return true;
};

import type { DiscordNotifier } from './DiscordNotifier';

export const validateDiscordNotifier = (isCreate: boolean, notifier: DiscordNotifier): boolean => {
  if (isCreate && !notifier.channelWebhookUrl) {
    return false;
  }

  return true;
};

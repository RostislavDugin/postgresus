import type { WebhookNotifier } from './WebhookNotifier';

export const validateWebhookNotifier = (isCreate: boolean, notifier: WebhookNotifier): boolean => {
  if (isCreate && !notifier.webhookUrl) {
    return false;
  }

  return true;
};

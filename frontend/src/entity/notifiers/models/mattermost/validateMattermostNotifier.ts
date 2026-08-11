import { MattermostDeliveryMode } from './MattermostDeliveryMode';
import type { MattermostNotifier } from './MattermostNotifier';

const MATTERMOST_CHANNEL_ID_LENGTH = 26;

export const validateMattermostNotifier = (
  isCreate: boolean,
  notifier: MattermostNotifier,
): boolean => {
  if (notifier.deliveryMode === MattermostDeliveryMode.WEBHOOK) {
    return !isCreate || !!notifier.webhookUrl;
  }

  if (notifier.deliveryMode === MattermostDeliveryMode.BOT) {
    if (!notifier.serverUrl) {
      return false;
    }

    if (isCreate && !notifier.botToken) {
      return false;
    }

    return notifier.targetChannelId.length === MATTERMOST_CHANNEL_ID_LENGTH;
  }

  return false;
};

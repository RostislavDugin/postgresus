import type { MattermostDeliveryMode } from './MattermostDeliveryMode';

export interface MattermostNotifier {
  deliveryMode: MattermostDeliveryMode;

  webhookUrl: string;

  serverUrl: string;
  botToken: string;

  targetChannelName: string;
  targetChannelId: string;

  overrideUsername: string;
  overrideIconUrl: string;

  isInsecureSkipVerify: boolean;
}

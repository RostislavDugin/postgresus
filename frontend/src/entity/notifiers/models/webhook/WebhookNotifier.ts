import type { WebhookHeader } from './WebhookHeader';
import type { WebhookMethod } from './WebhookMethod';

export interface WebhookNotifier {
  webhookUrl: string;
  webhookMethod: WebhookMethod;
  bodyTemplate?: string;
  headers?: WebhookHeader[];
}

import type { WebhookMethod } from './WebhookMethod';

export interface WebhookNotifier {
  webhookUrl: string;
  webhookMethod: WebhookMethod;
  customTemplate?: string; // 自定义模板，支持 {{heading}} 和 {{message}} 变量
}

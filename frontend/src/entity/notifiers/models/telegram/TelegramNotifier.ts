export interface TelegramNotifier {
  botToken: string;
  targetChatId: string;
  threadId?: number;
  isProxyEnabled?: boolean;
  proxyUrl?: string;
  botApiUrl?: string;

  // temp field
  isSendToThreadEnabled?: boolean;
}

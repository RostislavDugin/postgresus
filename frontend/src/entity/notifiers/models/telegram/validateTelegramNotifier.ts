import type { TelegramNotifier } from './TelegramNotifier';

const allowedProxyProtocols = ['http:', 'https:', 'socks5:', 'socks5h:'];

const isValidProxyUrl = (rawProxyUrl: string): boolean => {
  try {
    const proxyUrl = new URL(rawProxyUrl);
    return allowedProxyProtocols.includes(proxyUrl.protocol) && !!proxyUrl.host;
  } catch {
    return false;
  }
};

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

  if (notifier.isProxyEnabled) {
    if (isCreate && !notifier.proxyUrl) {
      return false;
    }

    if (notifier.proxyUrl && !isValidProxyUrl(notifier.proxyUrl)) {
      return false;
    }
  }

  return true;
};

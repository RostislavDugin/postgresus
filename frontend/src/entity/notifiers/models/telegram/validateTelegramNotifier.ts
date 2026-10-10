import type { TelegramNotifier } from './TelegramNotifier';

// eslint-disable-next-line i18next/no-literal-string -- URL protocols, not copy
const allowedProxyProtocols = ['http:', 'https:', 'socks5:', 'socks5h:'];

const isValidProxyUrl = (rawProxyUrl: string): boolean => {
  try {
    const proxyUrl = new URL(rawProxyUrl);
    return allowedProxyProtocols.includes(proxyUrl.protocol) && !!proxyUrl.host;
  } catch {
    return false;
  }
};

// eslint-disable-next-line i18next/no-literal-string -- URL protocols, not copy
const allowedBotApiProtocols = ['http:', 'https:'];

// Outside user info, which is rejected, "?" and "#" can only start a query or a fragment, and an
// empty one ("/telegram?") leaves search and hash empty.
const isValidBotApiUrl = (rawBotApiUrl: string): boolean => {
  try {
    const botApiUrl = new URL(rawBotApiUrl);
    return (
      allowedBotApiProtocols.includes(botApiUrl.protocol) &&
      !!botApiUrl.host &&
      !botApiUrl.username &&
      !botApiUrl.password &&
      !/[?#]/.test(rawBotApiUrl)
    );
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

  if (notifier.botApiUrl && !isValidBotApiUrl(notifier.botApiUrl)) {
    return false;
  }

  return true;
};

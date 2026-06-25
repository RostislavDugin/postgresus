import { describe, expect, it } from 'vitest';

import { validateTelegramNotifier } from './validateTelegramNotifier';

describe('validateTelegramNotifier', () => {
  it('requires proxy URL when proxy is enabled for a new notifier', () => {
    expect(
      validateTelegramNotifier(true, {
        botToken: 'token',
        targetChatId: '123456',
        isProxyEnabled: true,
      }),
    ).toBe(false);
  });

  it('allows an empty proxy URL on update (keeps the existing one)', () => {
    expect(
      validateTelegramNotifier(false, {
        botToken: '',
        targetChatId: '123456',
        isProxyEnabled: true,
      }),
    ).toBe(true);
  });

  it('rejects proxy URLs with an unsupported scheme', () => {
    expect(
      validateTelegramNotifier(true, {
        botToken: 'token',
        targetChatId: '123456',
        isProxyEnabled: true,
        proxyUrl: 'ftp://proxy.example.com:3128',
      }),
    ).toBe(false);
  });

  it('accepts http, https, socks5 and socks5h proxy URLs', () => {
    for (const proxyUrl of [
      'http://user:password@proxy.example.com:3128',
      'https://proxy.example.com:8443',
      'socks5://user:password@proxy.example.com:1080',
      'socks5h://proxy.example.com:1080',
    ]) {
      expect(
        validateTelegramNotifier(true, {
          botToken: 'token',
          targetChatId: '123456',
          isProxyEnabled: true,
          proxyUrl,
        }),
      ).toBe(true);
    }
  });
});

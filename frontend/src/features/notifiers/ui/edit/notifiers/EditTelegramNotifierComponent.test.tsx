import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  type Notifier,
  NotifierType,
  type TelegramNotifier,
} from '../../../../../entity/notifiers';
import { LocaleProvider } from '../../../../../shared/i18n';
import { EditTelegramNotifierComponent } from './EditTelegramNotifierComponent';

// The entity's public API imports its API client, and the client's config
// module reads window at import time. The tests run in node, so they supply
// window. LocaleProvider then reads localStorage, which here reports no saved
// language.
vi.hoisted(() => {
  vi.stubGlobal('window', {});
  vi.stubGlobal('localStorage', { getItem: () => null });
  vi.stubGlobal('navigator', { language: 'en-US', languages: ['en-US'] });
});

const createTelegramNotifier = (change: Partial<TelegramNotifier>): Notifier => ({
  id: 'notifier-id',
  name: 'Telegram',
  notifierType: NotifierType.TELEGRAM,
  workspaceId: 'workspace-id',
  telegramNotifier: {
    botToken: '',
    targetChatId: '-1001234567890',
    ...change,
  },
});

const renderTelegramForm = (notifier: Notifier) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <EditTelegramNotifierComponent
        notifier={notifier}
        setNotifier={() => {}}
        setUnsaved={() => {}}
      />
    </LocaleProvider>,
  );

describe('EditTelegramNotifierComponent', () => {
  it('starts with the advanced settings collapsed for a new notifier', () => {
    const markup = renderTelegramForm(createTelegramNotifier({}));

    expect(markup).toContain('Advanced settings');
    expect(markup).toContain('Target chat ID');
    expect(markup).not.toContain('Bot API URL');
    expect(markup).not.toContain('Use proxy');
  });

  it('starts with the advanced settings expanded when a proxy is enabled', () => {
    const markup = renderTelegramForm(createTelegramNotifier({ isProxyEnabled: true }));

    expect(markup).toContain('Bot API URL');
    expect(markup).toContain('Use proxy');
    expect(markup).toContain('Proxy URL');
  });

  it('starts with the advanced settings expanded when a Bot API URL is set', () => {
    const markup = renderTelegramForm(
      createTelegramNotifier({ botApiUrl: 'https://gateway.example.com/telegram' }),
    );

    expect(markup).toContain('https://gateway.example.com/telegram');
    expect(markup).toContain('Use proxy');
    expect(markup).not.toContain('Proxy URL');
  });
});

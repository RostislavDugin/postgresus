import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';

import {
  type Notifier,
  NotifierType,
  type TelegramNotifier,
} from '../../../../../entity/notifiers';
import { LocaleProvider } from '../../../../../shared/i18n';
import { ShowTelegramNotifierComponent } from './ShowTelegramNotifierComponent';

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

const renderTelegramView = (notifier: Notifier) =>
  renderToStaticMarkup(
    <LocaleProvider>
      <ShowTelegramNotifierComponent notifier={notifier} />
    </LocaleProvider>,
  );

describe('ShowTelegramNotifierComponent', () => {
  it('shows the Bot API URL when it is set', () => {
    const markup = renderTelegramView(
      createTelegramNotifier({ botApiUrl: 'https://gateway.example.com/telegram' }),
    );

    expect(markup).toContain('Bot API URL');
    expect(markup).toContain('https://gateway.example.com/telegram');
  });

  it('omits the Bot API URL when the official API is used', () => {
    const markup = renderTelegramView(createTelegramNotifier({}));

    expect(markup).not.toContain('Bot API URL');
  });
});

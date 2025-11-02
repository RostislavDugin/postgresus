import type { NotifierType } from './NotifierType';
import type { DiscordNotifier } from './discord/DiscordNotifier';
import type { EmailNotifier } from './email/EmailNotifier';
import type { SlackNotifier } from './slack/SlackNotifier';
import type { TeamsNotifier } from './teams/TeamsNotifier';
import type { TelegramNotifier } from './telegram/TelegramNotifier';
import type { WebhookNotifier } from './webhook/WebhookNotifier';

export interface Notifier {
  id: string;
  name: string;
  notifierType: NotifierType;
  lastSendError?: string;
  workspaceId: string;

  // specific notifier
  telegramNotifier?: TelegramNotifier;
  emailNotifier?: EmailNotifier;
  webhookNotifier?: WebhookNotifier;
  slackNotifier?: SlackNotifier;
  discordNotifier?: DiscordNotifier;
  teamsNotifier?: TeamsNotifier;
}

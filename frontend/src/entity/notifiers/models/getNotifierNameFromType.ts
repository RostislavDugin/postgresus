import { NotifierType } from './NotifierType';

export const getNotifierNameFromType = (type: NotifierType) => {
  switch (type) {
    case NotifierType.EMAIL:
      return 'Email';
    case NotifierType.TELEGRAM:
      return 'Telegram';
    case NotifierType.WEBHOOK:
      return 'Webhook';
    case NotifierType.SLACK:
      return 'Slack';
    case NotifierType.DISCORD:
      return 'Discord';
    case NotifierType.TEAMS:
      return 'Teams';
    case NotifierType.MATTERMOST:
      return 'Mattermost';
    default:
      return '';
  }
};

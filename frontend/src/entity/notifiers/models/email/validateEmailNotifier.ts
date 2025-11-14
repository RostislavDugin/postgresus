import type { EmailNotifier } from './EmailNotifier';

export const validateEmailNotifier = (isCreate: boolean, notifier: EmailNotifier): boolean => {
  if (!notifier.targetEmail) {
    return false;
  }

  if (!notifier.smtpHost) {
    return false;
  }

  if (!notifier.smtpPort) {
    return false;
  }

  if (isCreate && !notifier.smtpPassword) {
    return false;
  }

  return true;
};

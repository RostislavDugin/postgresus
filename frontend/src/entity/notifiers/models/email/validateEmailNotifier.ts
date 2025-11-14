import { FormValidator } from '../../../../shared/lib';
import type { EmailNotifier } from './EmailNotifier';

export const validateEmailNotifier = (isCreate: boolean, notifier: EmailNotifier): boolean => {
  if (!notifier.targetEmail || !FormValidator.isValidEmail(notifier.targetEmail)) {
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

  if (notifier.smtpUser && !FormValidator.isValidEmail(notifier.smtpUser)) {
    return false;
  }

  return true;
};

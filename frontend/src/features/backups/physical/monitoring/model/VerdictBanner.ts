import type { TranslationKey } from '../../../../../shared/i18n';
import type { MonitoringText } from './MonitoringText';
import type { MonitoringTone } from './MonitoringTone';
import type { VerdictIcon } from './VerdictIcon';

export interface VerdictBanner {
  tone: MonitoringTone;
  icon: VerdictIcon;
  headlineKey: TranslationKey;
  description?: MonitoringText;
  isConfigLinkShown: boolean;

  // False once refreshes have failed for too long to vouch for the verdict on screen.
  isStateConfirmed: boolean;
}

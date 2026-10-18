import type { WalStreamReasonLevel } from '../../../../../entity/backups/physical';
import type { TranslationKey } from '../../../../../shared/i18n';
import type { MonitoringText } from './MonitoringText';
import type { PathStage } from './PathStage';

export interface ReasonTexts {
  level: WalStreamReasonLevel;
  stage: PathStage;
  titleKey: TranslationKey;
  whatHappened: MonitoringText;
  whatToDo: MonitoringText;
}

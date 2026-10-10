import dayjs from 'dayjs';
import { useMemo } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { useLocale } from '../../../../shared/i18n';
import { getUserTimeFormat } from '../../../../shared/time';
import type { ScheduledBackupTime } from '../model/ScheduledBackupTime';
import { isScheduledBackupDue } from '../model/isScheduledBackupDue';

interface Props {
  scheduledBackupTime: ScheduledBackupTime;
  // a literal Tailwind class such as 'min-w-[150px]', aligning the row with the labels above
  labelWidthClassName: string;
}

export const NextScheduledBackupComponent = ({
  scheduledBackupTime,
  labelWidthClassName,
}: Props) => {
  const { t } = useTranslation();
  const { formatRelativeTime } = useLocale();

  const dateTimeFormat = useMemo(() => getUserTimeFormat(), []);

  const nextBackupText = isScheduledBackupDue(scheduledBackupTime) ? (
    t('backups.config.schedule.nextBackupDue')
  ) : (
    <Trans
      i18nKey="backups.config.schedule.nextBackup"
      values={{
        dateTime: dayjs(scheduledBackupTime.scheduledAt).local().format(dateTimeFormat.format),
        relativeTime: formatRelativeTime(scheduledBackupTime.scheduledAt),
      }}
      components={{ lineBreak: <br /> }}
    />
  );

  return (
    <div className="mb-1 flex w-full items-center text-xs text-gray-600 dark:text-gray-400">
      <div className={`${labelWidthClassName} pr-2`} />
      <div>{nextBackupText}</div>
    </div>
  );
};

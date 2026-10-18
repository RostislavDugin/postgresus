import { useTranslation } from 'react-i18next';

import { TransByKey, useLocale } from '../../../../../shared/i18n';
import { InlineCodeComponent } from '../../../../../shared/ui';
import { getMonitoringTextValues } from '../lib/monitoringTextValues';
import type { MonitoringText } from '../model/MonitoringText';

interface Props {
  text: MonitoringText;
  onOpenConfig?: () => void;
}

export const MonitoringTextComponent = ({ text, onOpenConfig }: Props) => {
  const { t } = useTranslation();
  const { formatNumber } = useLocale();

  return (
    <TransByKey
      i18nKey={text.key}
      values={getMonitoringTextValues(text, t, formatNumber)}
      components={{
        code: <InlineCodeComponent />,
        configLink: (
          <a
            href="#"
            className="underline"
            onClick={(event) => {
              event.preventDefault();
              onOpenConfig?.();
            }}
          />
        ),
      }}
    />
  );
};

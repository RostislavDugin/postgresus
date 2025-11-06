import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowTeamsNotifierComponent({ notifier }: Props) {
  const { t } = useTranslation(['notifier', 'common']);
  const url = notifier?.teamsNotifier?.powerAutomateUrl || '';
  const [expanded, setExpanded] = useState(false);

  const MAX = 20;
  const isLong = url.length > MAX;
  const display = expanded ? url : isLong ? `${url.slice(0, MAX)}…` : url;

  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">{t('notifier:form.teams_power_automate_url_label')}: </div>
        <div className="w-[250px] break-all">
          {url ? (
            <>
              <span title={url}>{display}</span>
              {isLong && (
                <button
                  type="button"
                  onClick={() => setExpanded((v) => !v)}
                  className="ml-2 text-xs text-blue-600 hover:underline"
                >
                  {expanded ? t('common:action.hide') : t('common:action.show')}
                </button>
              )}
            </>
          ) : (
            '—'
          )}
        </div>
      </div>
    </>
  );
}

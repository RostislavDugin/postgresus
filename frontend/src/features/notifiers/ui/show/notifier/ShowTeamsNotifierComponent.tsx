import { useState } from 'react';

import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowTeamsNotifierComponent({ notifier }: Props) {
  const url = notifier?.teamsNotifier?.powerAutomateUrl || '';
  const [expanded, setExpanded] = useState(false);

  const MAX = 20;
  const isLong = url.length > MAX;
  const display = expanded ? url : isLong ? `${url.slice(0, MAX)}…` : url;

  return (
    <>
      <div className="flex items-center">
        <div className="min-w-[110px]">Power Automate URL: </div>
        <div className="w-[50px] break-all md:w-[250px]">
          {url ? (
            <>
              <span title={url}>{display}</span>
              {isLong && (
                <button
                  type="button"
                  onClick={() => setExpanded((v) => !v)}
                  className="ml-2 text-xs text-blue-600 hover:underline"
                >
                  {expanded ? 'Hide' : 'Show'}
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

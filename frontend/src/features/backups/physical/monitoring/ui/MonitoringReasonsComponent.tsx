import { CloseCircleFilled, ExclamationCircleFilled } from '@ant-design/icons';
import { Alert, Collapse, theme } from 'antd';
import { useTranslation } from 'react-i18next';

import { WalStreamNoticeCode, WalStreamReasonLevel } from '../../../../../entity/backups/physical';
import type { ReasonTexts } from '../model/ReasonTexts';
import { MonitoringTextComponent } from './MonitoringTextComponent';

interface Props {
  reasons: ReasonTexts[];
  notices: { code: WalStreamNoticeCode; texts: ReasonTexts }[];
}

export const MonitoringReasonsComponent = ({ reasons, notices }: Props) => {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  if (reasons.length === 0 && notices.length === 0) {
    return null;
  }

  return (
    <div className="mt-4 space-y-3">
      {reasons.length > 0 && (
        <ul className="space-y-3">
          {reasons.map((reason) => {
            const isBroken = reason.level === WalStreamReasonLevel.BROKEN;

            return (
              <li key={reason.titleKey} className="flex items-start gap-3">
                <span
                  className="mt-0.5 text-lg leading-none"
                  style={{ color: isBroken ? token.colorError : token.colorWarning }}
                  aria-hidden
                >
                  {isBroken ? <CloseCircleFilled /> : <ExclamationCircleFilled />}
                </span>

                <div>
                  <div className="font-semibold">{t(reason.titleKey)}</div>
                  <div>
                    <MonitoringTextComponent text={reason.whatHappened} />
                  </div>
                  <div className="text-gray-600 dark:text-gray-400">
                    <MonitoringTextComponent text={reason.whatToDo} />
                  </div>
                </div>
              </li>
            );
          })}
        </ul>
      )}

      {notices.length > 0 && (
        <Alert
          type="info"
          showIcon
          message={
            <div className="space-y-2">
              {notices.map(({ code, texts }) =>
                code === WalStreamNoticeCode.NO_SLOT_WAL_LIMIT ? (
                  <Collapse
                    key={code}
                    ghost
                    size="small"
                    items={[
                      {
                        key: code,
                        label: (
                          <div>
                            <div className="font-semibold">{t(texts.titleKey)}</div>
                            <MonitoringTextComponent text={texts.whatHappened} />
                          </div>
                        ),
                        children: <MonitoringTextComponent text={texts.whatToDo} />,
                      },
                    ]}
                  />
                ) : (
                  <div key={code}>
                    <div className="font-semibold">{t(texts.titleKey)}</div>
                    <div>
                      <MonitoringTextComponent text={texts.whatHappened} />
                    </div>
                    <div className="text-gray-600 dark:text-gray-400">
                      <MonitoringTextComponent text={texts.whatToDo} />
                    </div>
                  </div>
                ),
              )}
            </div>
          }
        />
      )}
    </div>
  );
};

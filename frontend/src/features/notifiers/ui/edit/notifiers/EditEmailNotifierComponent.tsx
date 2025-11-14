import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';

import type { Notifier } from '../../../../../entity/notifiers';
import { FormValidator } from '../../../../../shared/lib';

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditEmailNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">Target email</div>
        <Input
          value={notifier?.emailNotifier?.targetEmail || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                targetEmail: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="example@gmail.com"
          status={
            notifier?.emailNotifier?.targetEmail &&
            !FormValidator.isValidEmail(notifier.emailNotifier.targetEmail)
              ? 'error'
              : undefined
          }
        />

        <Tooltip className="cursor-pointer" title="The email where you want to receive the message">
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">SMTP host</div>
        <Input
          value={notifier?.emailNotifier?.smtpHost || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpHost: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="smtp.gmail.com"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">SMTP port</div>
        <Input
          type="number"
          value={notifier?.emailNotifier?.smtpPort || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpPort: Number(e.target.value),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="25"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">SMTP user</div>
        <Input
          value={notifier?.emailNotifier?.smtpUser || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpUser: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="user@gmail.com"
          status={
            notifier?.emailNotifier?.smtpUser &&
            !FormValidator.isValidEmail(notifier.emailNotifier.smtpUser)
              ? 'error'
              : undefined
          }
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">SMTP password</div>
        <Input
          type="password"
          value={notifier?.emailNotifier?.smtpPassword || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                smtpPassword: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="password"
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">From</div>
        <Input
          value={notifier?.emailNotifier?.from || ''}
          onChange={(e) => {
            if (!notifier?.emailNotifier) return;

            setNotifier({
              ...notifier,
              emailNotifier: {
                ...notifier.emailNotifier,
                from: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder="example@example.com"
        />

        <Tooltip
          className="cursor-pointer"
          title="Optional. Email address to use as sender. If empty, will use SMTP user or auto-generate from host"
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>
    </>
  );
}

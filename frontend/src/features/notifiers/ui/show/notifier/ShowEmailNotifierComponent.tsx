import type { Notifier } from '../../../../../entity/notifiers';

interface Props {
  notifier: Notifier;
}

export function ShowEmailNotifierComponent({ notifier }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Target email</div>
        {notifier?.emailNotifier?.targetEmail}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">SMTP host</div>
        {notifier?.emailNotifier?.smtpHost}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">SMTP port</div>
        {notifier?.emailNotifier?.smtpPort}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">SMTP user</div>
        {notifier?.emailNotifier?.smtpUser}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">SMTP password</div>
        {'*************'}
      </div>
    </>
  );
}

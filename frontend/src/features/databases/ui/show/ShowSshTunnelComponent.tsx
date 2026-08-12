import type { SshTunnelConfig } from '../../../../entity/databases';

interface Props {
  sshTunnel: SshTunnelConfig | undefined;
}

export const ShowSshTunnelComponent = ({ sshTunnel }: Props) => {
  if (!sshTunnel?.isEnabled) return null;

  return (
    <>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">SSH host</div>
        <div>{sshTunnel.host}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">SSH port</div>
        <div>{sshTunnel.port}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">SSH username</div>
        <div>{sshTunnel.username}</div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">SSH credentials</div>
        <div>*************</div>
      </div>
    </>
  );
};

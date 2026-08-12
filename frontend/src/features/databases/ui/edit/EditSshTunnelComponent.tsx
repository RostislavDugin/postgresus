import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Checkbox, Input, InputNumber, Tooltip } from 'antd';
import { useState } from 'react';

import {
  DEFAULT_SSH_PORT,
  type SshTunnelConfig,
  createEmptySshTunnelConfig,
} from '../../../../entity/databases';

interface Props {
  sshTunnel: SshTunnelConfig | undefined;
  hasStoredSecrets: boolean;
  onChange: (sshTunnel: SshTunnelConfig) => void;
}

export const EditSshTunnelComponent = ({ sshTunnel, hasStoredSecrets, onChange }: Props) => {
  const [isReplacingSecrets, setIsReplacingSecrets] = useState(false);

  const updateField = <Field extends keyof SshTunnelConfig>(
    field: Field,
    value: SshTunnelConfig[Field],
  ) => {
    onChange({ ...currentTunnel, [field]: value });
  };

  const startReplacingSecrets = () => {
    setIsReplacingSecrets(true);
    onChange({ ...currentTunnel, password: '', privateKey: '', privateKeyPassphrase: '' });
  };

  const renderCredentials = () => {
    if (hasStoredSecrets && !isReplacingSecrets) {
      return (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">SSH credentials</div>
          <div className="flex items-center">
            <span className="mr-3">*************</span>
            <Button size="small" onClick={startReplacingSecrets}>
              Replace
            </Button>
          </div>
        </div>
      );
    }

    return (
      <>
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">SSH password</div>
          <Input.Password
            value={currentTunnel.password}
            onChange={(e) => updateField('password', e.target.value)}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Leave empty when using a key"
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
        </div>

        <div className="mb-1 flex w-full items-start">
          <div className="min-w-[150px]">SSH private key</div>
          <Input.TextArea
            value={currentTunnel.privateKey}
            onChange={(e) => updateField('privateKey', e.target.value)}
            size="small"
            className="max-w-[300px] grow"
            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
            autoSize={{ minRows: 2, maxRows: 5 }}
          />
        </div>

        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">Key passphrase</div>
          <Input.Password
            value={currentTunnel.privateKeyPassphrase}
            onChange={(e) => updateField('privateKeyPassphrase', e.target.value)}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Only for an encrypted key"
            autoComplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
          />
        </div>
      </>
    );
  };

  const currentTunnel = sshTunnel ?? createEmptySshTunnelConfig();

  return (
    <>
      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]" />
        <Checkbox
          checked={currentTunnel.isEnabled}
          onChange={(e) => updateField('isEnabled', e.target.checked)}
        >
          <div className="flex items-center">
            <span>Connect through an SSH tunnel</span>
            <Tooltip
              className="cursor-pointer"
              title="For a database inside a closed network. Databasus connects to the SSH host below, and that host reaches the database using the host and port above."
            >
              <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
            </Tooltip>
          </div>
        </Checkbox>
      </div>

      {currentTunnel.isEnabled && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">SSH host</div>
            <Input
              value={currentTunnel.host}
              onChange={(e) => updateField('host', e.target.value)}
              size="small"
              className="max-w-[200px] grow"
              placeholder="bastion.example.com"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]" />
            <div className="text-xs text-gray-500">
              The database host above is resolved by the SSH host, not by Databasus. Use 127.0.0.1
              when the database runs on the SSH host itself.
            </div>
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">SSH port</div>
            <InputNumber
              value={currentTunnel.port}
              onChange={(value) => updateField('port', value ?? DEFAULT_SSH_PORT)}
              size="small"
              className="max-w-[200px] grow"
              min={1}
              max={65535}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">SSH username</div>
            <Input
              value={currentTunnel.username}
              onChange={(e) => updateField('username', e.target.value)}
              size="small"
              className="max-w-[200px] grow"
              placeholder="Enter SSH username"
            />
          </div>

          {renderCredentials()}
        </>
      )}
    </>
  );
};

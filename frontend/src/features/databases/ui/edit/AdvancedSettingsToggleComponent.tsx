import { DownOutlined, UpOutlined } from '@ant-design/icons';

interface Props {
  isShowAdvanced: boolean;
  onToggle: () => void;
}

export const AdvancedSettingsToggleComponent = ({ isShowAdvanced, onToggle }: Props) => (
  <div className="mt-4 mb-1 flex items-center">
    <div
      className="flex cursor-pointer items-center text-sm text-blue-600 hover:text-blue-800"
      onClick={onToggle}
    >
      <span className="mr-2">Advanced settings</span>

      {isShowAdvanced ? (
        <UpOutlined style={{ fontSize: '12px' }} />
      ) : (
        <DownOutlined style={{ fontSize: '12px' }} />
      )}
    </div>
  </div>
);

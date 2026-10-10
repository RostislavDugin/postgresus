import { InfoCircleOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import type { ReactNode } from 'react';

interface Props {
  label: string;
  tooltip: string;
  children: ReactNode;
}

export const AdvancedFieldRowComponent = ({ label, tooltip, children }: Props) => (
  <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
    <div className="mb-1 min-w-[150px] sm:mb-0 sm:pr-2">{label}</div>
    <div className="flex items-center">
      {children}

      <Tooltip className="cursor-pointer" title={tooltip}>
        <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
      </Tooltip>
    </div>
  </div>
);

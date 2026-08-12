import { Tooltip } from 'antd';

interface Props {
  className: string;
}

export const SponsorshipLinkComponent = ({ className }: Props) => {
  return (
    <Tooltip title="Databasus saved your data? You can support Databasus as well to keep it free, maintained and independent">
      <a
        className={`!underline !decoration-blue-600 !decoration-2 underline-offset-4 ${className}`}
        href="https://databasus.com/sponsorship"
        target="_blank"
        rel="noreferrer"
      >
        Sponsorship
      </a>
    </Tooltip>
  );
};

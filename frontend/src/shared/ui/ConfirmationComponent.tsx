import { Button, Modal } from 'antd';
import type { JSX } from 'react';
import { useTranslation } from 'react-i18next';

interface Props {
  onConfirm(): void;
  onDecline(): void;

  description: string;
  actionButtonColor: 'blue' | 'red';

  actionText: string;
  cancelText?: string;
  hideCancelButton?: boolean;
}

export function ConfirmationComponent({
  onConfirm,
  onDecline,
  description,
  actionButtonColor,
  actionText,
  cancelText,
  hideCancelButton = false,
}: Props): JSX.Element {
  const { t } = useTranslation(['common']);

  return (
    <Modal
      title={t('common:common.confirmation')}
      open
      onClose={() => onDecline()}
      onCancel={() => onDecline()}
      footer={<div />}
    >
      <div dangerouslySetInnerHTML={{ __html: description }} />

      <div className="mt-5 flex">
        {!hideCancelButton && (
          <Button
            className="ml-auto"
            onClick={() => onDecline()}
            danger={actionButtonColor !== 'red'}
            type="primary"
          >
            {cancelText || t('common:common.cancel')}
          </Button>
        )}

        <Button
          className="ml-1"
          onClick={() => onConfirm()}
          danger={actionButtonColor === 'red'}
          type="primary"
        >
          {actionText}
        </Button>
      </div>
    </Modal>
  );
}

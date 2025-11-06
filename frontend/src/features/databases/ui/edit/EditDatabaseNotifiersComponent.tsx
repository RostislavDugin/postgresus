import { Button, Modal, Select, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, databaseApi } from '../../../../entity/databases';
import { type Notifier, notifierApi } from '../../../../entity/notifiers';
import { EditNotifierComponent } from '../../../notifiers/ui/edit/EditNotifierComponent';

interface Props {
  database: Database;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowSaveOnlyForUnsaved: boolean;
  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;
}

export const EditDatabaseNotifiersComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  isShowSaveOnlyForUnsaved,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const { t } = useTranslation(['database', 'common']);
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const [notifiers, setNotifiers] = useState<Notifier[]>([]);
  const [isNotifiersLoading, setIsNotifiersLoading] = useState(false);
  const [isShowCreateNotifier, setShowCreateNotifier] = useState(false);

  const saveDatabase = async () => {
    if (!editingDatabase) return;

    if (isSaveToApi) {
      setIsSaving(true);

      try {
        await databaseApi.updateDatabase(editingDatabase);
        setIsUnsaved(false);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }

    onSaved(editingDatabase);
  };

  const loadNotifiers = async () => {
    setIsNotifiersLoading(true);

    try {
      const notifiers = await notifierApi.getNotifiers();
      setNotifiers(notifiers);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsNotifiersLoading(false);
  };

  useEffect(() => {
    setIsSaving(false);
    setEditingDatabase({ ...database });
    loadNotifiers();
  }, [database]);

  if (!editingDatabase) return null;

  if (isNotifiersLoading)
    return (
      <div className="mb-5 flex items-center">
        <Spin />
      </div>
    );

  return (
    <div>
      <div className="mb-5 max-w-[275px] text-gray-500">
        {t('database:notifiers.description_line1')}
        <br />
        <br />
        {t('database:notifiers.description_line2')}
      </div>

      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px]">{t('database:notifiers.label')}</div>

        <Select
          mode="multiple"
          value={editingDatabase.notifiers.map((n) => n.id)}
          onChange={(notifiersIds) => {
            if (notifiersIds.includes('create-new-notifier')) {
              setShowCreateNotifier(true);
              return;
            }

            setEditingDatabase({
              ...editingDatabase,
              notifiers: notifiers.filter((n) => notifiersIds.includes(n.id)),
            } as unknown as Database);

            setIsUnsaved(true);
          }}
          size="small"
          className="max-w-[200px] grow"
          options={[
            ...notifiers.map((n) => ({ label: n.name, value: n.id })),
            { label: t('database:notifiers.create_new_notifier'), value: 'create-new-notifier' },
          ]}
          placeholder={t('database:notifiers.select_placeholder')}
        />
      </div>

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button className="mr-1" danger ghost onClick={() => onCancel()}>
            {t('common:common.cancel')}
          </Button>
        )}

        {isShowBackButton && (
          <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
            {t('common:common.back')}
          </Button>
        )}

        {(!isShowSaveOnlyForUnsaved || isUnsaved) && (
          <Button
            type="primary"
            onClick={() => saveDatabase()}
            loading={isSaving}
            disabled={isSaving}
            className="mr-5"
          >
            {saveButtonText || t('common:common.save')}
          </Button>
        )}
      </div>

      {isShowCreateNotifier && (
        <Modal
          title={t('database:notifiers.modal_title')}
          footer={<div />}
          open={isShowCreateNotifier}
          onCancel={() => setShowCreateNotifier(false)}
        >
          <div className="my-3 max-w-[275px] text-gray-500">
            {t('database:notifiers.description_line1')}
          </div>

          <EditNotifierComponent
            isShowName
            isShowClose={false}
            onClose={() => setShowCreateNotifier(false)}
            onChanged={() => {
              loadNotifiers();
              setShowCreateNotifier(false);
            }}
          />
        </Modal>
      )}
    </div>
  );
};

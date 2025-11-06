import { Button, Input } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Database, databaseApi } from '../../../../entity/databases';

interface Props {
  database: Database;

  isShowName?: boolean;
  isShowCancelButton?: boolean;
  onCancel: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (db: Database) => void;
}

export const EditDatabaseBaseInfoComponent = ({
  database,
  isShowName,
  isShowCancelButton,
  onCancel,
  saveButtonText,
  isSaveToApi,
  onSaved,
}: Props) => {
  const { t } = useTranslation(['database', 'common']);
  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isUnsaved, setIsUnsaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const updateDatabase = (patch: Partial<Database>) => {
    setEditingDatabase((prev) => (prev ? { ...prev, ...patch } : prev));
    setIsUnsaved(true);
  };

  const saveDatabase = async () => {
    if (!editingDatabase) return;
    if (isSaveToApi) {
      setIsSaving(true);

      try {
        editingDatabase.name = editingDatabase.name?.trim();
        await databaseApi.updateDatabase(editingDatabase);
        setIsUnsaved(false);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }
    onSaved(editingDatabase);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsUnsaved(false);
    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  // mandatory-field check
  const isAllFieldsFilled = !!editingDatabase.name?.trim();

  return (
    <div>
      {isShowName && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">{t('database:form.name_label')}</div>
          <Input
            value={editingDatabase.name || ''}
            onChange={(e) => updateDatabase({ name: e.target.value })}
            size="small"
            placeholder={t('database:form.name_placeholder')}
            className="max-w-[200px] grow"
          />
        </div>
      )}

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button danger ghost className="mr-1" onClick={onCancel}>
            {t('common:button.cancel')}
          </Button>
        )}

        <Button
          type="primary"
          className={`${isShowCancelButton ? 'ml-1' : 'ml-auto'} mr-5`}
          onClick={saveDatabase}
          loading={isSaving}
          disabled={(isSaveToApi && !isUnsaved) || !isAllFieldsFilled}
        >
          {saveButtonText || t('common:button.save')}
        </Button>
      </div>
    </div>
  );
};

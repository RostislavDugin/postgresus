import { type Database, DatabaseType } from '../../../../entity/databases';
import { EditMySqlSpecificDataComponent } from './EditMySqlSpecificDataComponent';
import { EditPostgreSqlSpecificDataComponent } from './EditPostgreSqlSpecificDataComponent';

interface Props {
  database: Database;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;

  isShowDbName?: boolean;
  isRestoreMode?: boolean;
}

export const EditDatabaseSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
  isRestoreMode = false,
}: Props) => {
  if (database.type === DatabaseType.POSTGRES) {
    return (
      <EditPostgreSqlSpecificDataComponent
        database={database}
        isShowCancelButton={isShowCancelButton}
        onCancel={onCancel}
        isShowBackButton={isShowBackButton}
        onBack={onBack}
        saveButtonText={saveButtonText}
        isSaveToApi={isSaveToApi}
        onSaved={onSaved}
        isShowDbName={isShowDbName}
        isRestoreMode={isRestoreMode}
      />
    );
  }

  if (database.type === DatabaseType.MYSQL) {
    return (
      <EditMySqlSpecificDataComponent
        database={database}
        isShowCancelButton={isShowCancelButton}
        onCancel={onCancel}
        isShowBackButton={isShowBackButton}
        onBack={onBack}
        saveButtonText={saveButtonText}
        isSaveToApi={isSaveToApi}
        onSaved={onSaved}
        isShowDbName={isShowDbName}
      />
    );
  }

  return null;
};

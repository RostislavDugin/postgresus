import { type Database, DatabaseType } from '../../../../entity/databases';
import { EditMariaDbSpecificDataComponent } from './EditMariaDbSpecificDataComponent';
import { EditMongoDbSpecificDataComponent } from './EditMongoDbSpecificDataComponent';
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
  const commonProps = {
    database,
    isShowCancelButton,
    onCancel,
    isShowBackButton,
    onBack,
    saveButtonText,
    isSaveToApi,
    onSaved,
    isShowDbName,
  };

  switch (database.type) {
    case DatabaseType.POSTGRES:
      return <EditPostgreSqlSpecificDataComponent {...commonProps} isRestoreMode={isRestoreMode} />;
    case DatabaseType.MYSQL:
      return <EditMySqlSpecificDataComponent {...commonProps} />;
    case DatabaseType.MARIADB:
      return <EditMariaDbSpecificDataComponent {...commonProps} />;
    case DatabaseType.MONGODB:
      return <EditMongoDbSpecificDataComponent {...commonProps} />;
    default:
      return null;
  }
};

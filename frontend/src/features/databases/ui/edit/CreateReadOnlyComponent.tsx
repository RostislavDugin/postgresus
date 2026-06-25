import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, DatabaseType, databaseApi } from '../../../../entity/databases';

interface Props {
  database: Database;
  onReadOnlyUserUpdated: (database: Database) => void;

  onGoBack: () => void;
  onSkipped: () => void;
  onAlreadyExists: () => void;
}

const PRIVILEGES_TRUNCATE_LENGTH = 50;

export const CreateReadOnlyComponent = ({
  database,
  onReadOnlyUserUpdated,
  onGoBack,
  onSkipped,
  onAlreadyExists,
}: Props) => {
  const [isCheckingReadOnlyUser, setIsCheckingReadOnlyUser] = useState(false);
  const [isCreatingReadOnlyUser, setIsCreatingReadOnlyUser] = useState(false);
  const [isShowSkipConfirmation, setShowSkipConfirmation] = useState(false);
  const [privileges, setPrivileges] = useState<string[]>([]);
  const [isPrivilegesExpanded, setIsPrivilegesExpanded] = useState(false);

  const isPostgres = database.type === DatabaseType.POSTGRES_LOGICAL;
  const isPhysicalPostgres = database.type === DatabaseType.POSTGRES_PHYSICAL;
  const isMysql = database.type === DatabaseType.MYSQL;
  const isMariadb = database.type === DatabaseType.MARIADB;
  const isMongodb = database.type === DatabaseType.MONGODB;
  const databaseTypeName =
    isPostgres || isPhysicalPostgres
      ? 'PostgreSQL'
      : isMysql
        ? 'MySQL'
        : isMariadb
          ? 'MariaDB'
          : isMongodb
            ? 'MongoDB'
            : 'database';

  const privilegesLabel = isMongodb ? 'roles' : 'privileges';
  const userKindNoun = isPhysicalPostgres ? 'replication-only user' : 'read-only user';

  const checkReadOnlyUser = async (): Promise<boolean> => {
    try {
      const response = await databaseApi.isUserReadOnly(database);
      setPrivileges(response.privileges || []);
      return response.isReadOnly;
    } catch (e) {
      alert((e as Error).message);
      return false;
    }
  };

  const getPrivilegesDisplay = () => {
    const fullText = privileges.join(', ');
    if (isPrivilegesExpanded || fullText.length <= PRIVILEGES_TRUNCATE_LENGTH) {
      return fullText;
    }

    return fullText.substring(0, PRIVILEGES_TRUNCATE_LENGTH) + '...';
  };

  const shouldShowExpandToggle = () => {
    const fullText = privileges.join(', ');
    return fullText.length > PRIVILEGES_TRUNCATE_LENGTH;
  };

  const createReadOnlyUser = async () => {
    setIsCreatingReadOnlyUser(true);

    try {
      const response = isPhysicalPostgres
        ? await databaseApi.createReplicationOnlyUser(database)
        : await databaseApi.createReadOnlyUser(database);

      if (isPhysicalPostgres && database.postgresqlPhysical) {
        database.postgresqlPhysical.username = response.username;
        database.postgresqlPhysical.password = response.password;
      } else if (isPostgres && database.postgresqlLogical) {
        database.postgresqlLogical.username = response.username;
        database.postgresqlLogical.password = response.password;
      } else if (isMysql && database.mysql) {
        database.mysql.username = response.username;
        database.mysql.password = response.password;
      } else if (isMariadb && database.mariadb) {
        database.mariadb.username = response.username;
        database.mariadb.password = response.password;
      } else if (isMongodb && database.mongodb) {
        database.mongodb.username = response.username;
        database.mongodb.password = response.password;
      }

      onReadOnlyUserUpdated(database);
    } catch (e) {
      alert((e as Error).message);
    }

    setIsCreatingReadOnlyUser(false);
  };

  const handleSkip = () => {
    setShowSkipConfirmation(true);
  };

  const handleSkipConfirmed = () => {
    setShowSkipConfirmation(false);
    onSkipped();
  };

  useEffect(() => {
    const run = async () => {
      setIsCheckingReadOnlyUser(true);

      const isReadOnly = await checkReadOnlyUser();
      if (isReadOnly) {
        onAlreadyExists();
      }

      setIsCheckingReadOnlyUser(false);
    };
    run();
  }, []);

  if (isCheckingReadOnlyUser) {
    return (
      <div className="flex items-center">
        <Spin />
        <span className="ml-3">Checking {userKindNoun}...</span>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-5">
        <p className="mb-3 text-lg font-bold">Create a {userKindNoun} for Databasus?</p>

        <p className="mb-2">
          A {userKindNoun} is a {databaseTypeName} user with limited permissions that can only read
          data from your database, not modify it. This is recommended for backup operations because:
        </p>

        <ul className="mb-2 ml-5 list-disc">
          <li>it prevents accidental data modifications during backup</li>
          <li>it follows the principle of least privilege</li>
          <li>it&apos;s a security best practice</li>
        </ul>

        <p className="mb-2">
          Databasus enforce enterprise-grade security (
          <a
            href="https://databasus.com/security"
            target="_blank"
            rel="noreferrer"
            className="!text-blue-600 dark:!text-blue-400"
          >
            read in details here
          </a>
          ). However, it is not possible to be covered from all possible risks.
        </p>

        <p className="mt-3">
          <b>A {userKindNoun} allows to avoid storing credentials with write access at all</b>. Even
          in the worst case of hacking, nobody will be able to corrupt your data.
        </p>

        <p className="mt-3">
          {privileges.length === 0 ? (
            <>
              Current user has <b>no write {privilegesLabel}</b>.
            </>
          ) : (
            <>
              Current user has the following write {privilegesLabel}:{' '}
              <span
                className={shouldShowExpandToggle() ? 'cursor-pointer hover:opacity-80' : ''}
                onClick={() =>
                  shouldShowExpandToggle() && setIsPrivilegesExpanded(!isPrivilegesExpanded)
                }
              >
                {getPrivilegesDisplay()}
                {shouldShowExpandToggle() && (
                  <span className="ml-1 text-xs text-blue-600 hover:opacity-80">
                    ({isPrivilegesExpanded ? 'collapse' : 'expand'})
                  </span>
                )}
              </span>
            </>
          )}
        </p>
      </div>

      <div className="mt-5 flex">
        <Button className="mr-auto" type="primary" ghost onClick={() => onGoBack()}>
          Back
        </Button>

        <Button className="mr-2 ml-auto" danger ghost onClick={handleSkip}>
          Skip
        </Button>

        <Button
          type="primary"
          onClick={createReadOnlyUser}
          loading={isCreatingReadOnlyUser}
          disabled={isCreatingReadOnlyUser}
        >
          Yes, create {userKindNoun}
        </Button>
      </div>

      <Modal
        title={`Skip ${userKindNoun} creation?`}
        open={isShowSkipConfirmation}
        onCancel={() => setShowSkipConfirmation(false)}
        footer={null}
        width={450}
      >
        <div className="mb-5">
          <p className="mb-2">Are you sure you want to skip creating a {userKindNoun}?</p>

          <p className="mb-2">
            Using a user with full permissions for backups is not recommended and may pose security
            risks. Databasus is highly recommending you to not skip this step.
          </p>

          <p>
            100% protection is never possible. It&apos;s better to be safe in case of 0.01% risk of
            full hacking. So it is better to follow the secure way with read-only user.
          </p>
        </div>

        <div className="flex justify-end">
          <Button className="mr-2" danger ghost onClick={handleSkipConfirmed}>
            Yes, I accept risks
          </Button>

          <Button type="primary" onClick={() => setShowSkipConfirmation(false)}>
            Let&apos;s continue with the secure way
          </Button>
        </div>
      </Modal>
    </div>
  );
};

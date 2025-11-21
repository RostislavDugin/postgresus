import { Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, databaseApi } from '../../../../entity/databases';

interface Props {
  database: Database;
  onReadOnlyUserUpdated: (database: Database) => void;

  onGoBack: () => void;
  onContinue: () => void;
}

export const CreateReadOnlyComponent = ({
  database,
  onReadOnlyUserUpdated,
  onGoBack,
  onContinue,
}: Props) => {
  const [isCheckingReadOnlyUser, setIsCheckingReadOnlyUser] = useState(false);
  const [isCreatingReadOnlyUser, setIsCreatingReadOnlyUser] = useState(false);
  const [isShowSkipConfirmation, setShowSkipConfirmation] = useState(false);

  const checkReadOnlyUser = async (): Promise<boolean> => {
    try {
      const response = await databaseApi.isUserReadOnly(database);
      return response.isReadOnly;
    } catch (e) {
      alert((e as Error).message);
      return false;
    }
  };

  const createReadOnlyUser = async () => {
    setIsCreatingReadOnlyUser(true);

    try {
      const response = await databaseApi.createReadOnlyUser(database);
      database.postgresql!.username = response.username;
      database.postgresql!.password = response.password;
      onReadOnlyUserUpdated(database);
      onContinue();
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
    onContinue();
  };

  useEffect(() => {
    const run = async () => {
      setIsCheckingReadOnlyUser(true);

      const isReadOnly = await checkReadOnlyUser();
      if (isReadOnly) {
        // already has a read-only user
        onContinue();
      }

      setIsCheckingReadOnlyUser(false);
    };
    run();
  }, []);

  if (isCheckingReadOnlyUser) {
    return (
      <div className="flex items-center">
        <Spin />
        <span className="ml-3">Checking read-only user...</span>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-5">
        <p className="mb-3 text-lg font-bold">Create a read-only user for Postgresus?</p>

        <p className="mb-2">
          A read-only user is a PostgreSQL user with limited permissions that can only read data
          from your database, not modify it. This is recommended for backup operations because:
        </p>

        <ul className="mb-2 ml-5 list-disc">
          <li>it prevents accidental data modifications during backup</li>
          <li>it follows the principle of least privilege</li>
          <li>it&apos;s a security best practice</li>
        </ul>

        <p className="mb-2">
          Postgresus enforce enterprise-grade security (
          <a href="https://postgresus.com/security" target="_blank" rel="noreferrer">
            read in details here
          </a>
          ). However, it is not possible to be covered from all possible risks.
        </p>

        <p className="mt-3">
          <b>Read-only user allows to avoid storing credentials with write access at all</b>. Even
          in the worst case of hacking, nobody will be able to corrupt your data.
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
          Yes, create read-only user
        </Button>
      </div>

      <Modal
        title="Skip read-only user creation?"
        open={isShowSkipConfirmation}
        onCancel={() => setShowSkipConfirmation(false)}
        footer={null}
        width={450}
      >
        <div className="mb-5">
          <p className="mb-2">Are you sure you want to skip creating a read-only user?</p>

          <p className="mb-2">
            Using a user with full permissions for backups is not recommended and may pose security
            risks. Postgresus is highly recommending you to not skip this step.
          </p>

          <p>
            100% protection is never possible. It&apos;s better to be safe in case of 0.01% risk of
            full hacking. So it is better to follow the secure way with read-only user.
          </p>
        </div>

        <div className="flex justify-end">
          <Button className="mr-2" danger onClick={handleSkipConfirmed}>
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

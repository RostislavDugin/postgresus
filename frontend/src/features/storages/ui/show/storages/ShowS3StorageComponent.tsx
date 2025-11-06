import { useTranslation } from 'react-i18next';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowS3StorageComponent({ storage }: Props) {
  const { t } = useTranslation(['storage']);

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.s3_bucket_label')}</div>
        {storage?.s3Storage?.s3Bucket}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.s3_region_label')}</div>
        {storage?.s3Storage?.s3Region}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.s3_access_key_label')}</div>
        {storage?.s3Storage?.s3AccessKey ? '*********' : ''}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.s3_secret_key_label')}</div>
        {storage?.s3Storage?.s3SecretKey ? '*********' : ''}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.s3_endpoint_label')}</div>
        {storage?.s3Storage?.s3Endpoint || '-'}
      </div>
    </>
  );
}

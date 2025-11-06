import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditS3StorageComponent({ storage, setStorage, setIsUnsaved }: Props) {
  const { t } = useTranslation(['storage']);

  return (
    <>
      <div className="mb-2 flex items-center">
        <div className="min-w-[110px]" />

        <div className="text-xs text-blue-600">
          <a href="https://postgresus.com/cloudflare-r2-storage" target="_blank" rel="noreferrer">
            {t('storage:form.cloudflare_r2_link')}
          </a>
        </div>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.s3_bucket_label')}</div>
        <Input
          value={storage?.s3Storage?.s3Bucket || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Bucket: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.s3_bucket_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.region_label')}</div>
        <Input
          value={storage?.s3Storage?.s3Region || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Region: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.region_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.access_key_label')}</div>
        <Input.Password
          value={storage?.s3Storage?.s3AccessKey || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3AccessKey: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.access_key_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.secret_key_label')}</div>
        <Input.Password
          value={storage?.s3Storage?.s3SecretKey || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3SecretKey: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.secret_key_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.endpoint_label')}</div>
        <Input
          value={storage?.s3Storage?.s3Endpoint || ''}
          onChange={(e) => {
            if (!storage?.s3Storage) return;

            setStorage({
              ...storage,
              s3Storage: {
                ...storage.s3Storage,
                s3Endpoint: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.endpoint_placeholder')}
        />

        <Tooltip
          className="cursor-pointer"
          title={t('storage:form.endpoint_tooltip')}
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>
    </>
  );
}

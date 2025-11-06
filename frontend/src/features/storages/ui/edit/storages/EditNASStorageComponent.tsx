import { InfoCircleOutlined } from '@ant-design/icons';
import { Input, InputNumber, Switch, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
  setStorage: (storage: Storage) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditNASStorageComponent({ storage, setStorage, setIsUnsaved }: Props) {
  const { t } = useTranslation(['storage']);

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.host_label')}</div>
        <Input
          value={storage?.nasStorage?.host || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                host: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.host_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.port_label')}</div>
        <InputNumber
          value={storage?.nasStorage?.port}
          onChange={(value) => {
            if (!storage?.nasStorage || !value) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                port: value,
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          min={1}
          max={65535}
          placeholder={t('storage:form.port_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.share_label')}</div>
        <Input
          value={storage?.nasStorage?.share || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                share: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.share_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.username_label')}</div>
        <Input
          value={storage?.nasStorage?.username || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                username: e.target.value.trim(),
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.username_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.password_label')}</div>
        <Input.Password
          value={storage?.nasStorage?.password || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                password: e.target.value,
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.password_placeholder')}
        />
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.use_ssl_label')}</div>
        <Switch
          checked={storage?.nasStorage?.useSsl || false}
          onChange={(checked) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                useSsl: checked,
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
        />

        <Tooltip className="cursor-pointer" title={t('storage:form.use_ssl_tooltip')}>
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.domain_label')}</div>
        <Input
          value={storage?.nasStorage?.domain || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                domain: e.target.value.trim() || undefined,
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.domain_placeholder')}
        />

        <Tooltip
          className="cursor-pointer"
          title={t('storage:form.domain_tooltip')}
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.path_label')}</div>
        <Input
          value={storage?.nasStorage?.path || ''}
          onChange={(e) => {
            if (!storage?.nasStorage) return;

            let pathValue = e.target.value.trim();
            // Remove leading slash if present
            if (pathValue.startsWith('/')) {
              pathValue = pathValue.substring(1);
            }

            setStorage({
              ...storage,
              nasStorage: {
                ...storage.nasStorage,
                path: pathValue || undefined,
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
          className="w-full max-w-[250px]"
          placeholder={t('storage:form.nas_path_placeholder')}
        />

        <Tooltip className="cursor-pointer" title={t('storage:form.nas_path_tooltip')}>
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>
    </>
  );
}

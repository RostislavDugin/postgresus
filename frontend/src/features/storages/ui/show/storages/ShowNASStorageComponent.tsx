import { useTranslation } from 'react-i18next';

import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowNASStorageComponent({ storage }: Props) {
  const { t } = useTranslation(['storage', 'common']);

  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_host_label')}</div>
        {storage?.nasStorage?.host || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_port_label')}</div>
        {storage?.nasStorage?.port || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_share_label')}</div>
        {storage?.nasStorage?.share || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_username_label')}</div>
        {storage?.nasStorage?.username || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_password_label')}</div>
        {storage?.nasStorage?.password ? '*********' : '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_use_ssl_label')}</div>
        {storage?.nasStorage?.useSsl ? t('common:common.yes') : t('common:common.no')}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_domain_label')}</div>
        {storage?.nasStorage?.domain || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">{t('storage:form.nas_path_label')}</div>
        {storage?.nasStorage?.path || '-'}
      </div>
    </>
  );
}

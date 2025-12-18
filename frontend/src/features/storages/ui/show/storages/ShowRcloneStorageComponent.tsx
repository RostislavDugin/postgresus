import type { Storage } from '../../../../../entity/storages';

interface Props {
  storage: Storage;
}

export function ShowRcloneStorageComponent({ storage }: Props) {
  return (
    <>
      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Remote path</div>
        {storage?.rcloneStorage?.remotePath || '-'}
      </div>

      <div className="mb-1 flex items-center">
        <div className="min-w-[110px]">Config</div>
        {'*************'}
      </div>
    </>
  );
}

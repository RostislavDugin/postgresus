import { App, Button, Checkbox, Input, InputNumber, Modal, Select, Spin, Switch, TimePicker } from 'antd';
import dayjs, { Dayjs } from 'dayjs';
import { useEffect, useState } from 'react';

import type { WorkspaceResponse } from '../../../entity/workspaces';
import { clusterApi } from '../../../entity/clusters/api/clusterApi';
import type { ApplyPropagationRequest, PropagationChange } from '../../../entity/clusters/api/clusterApi';
import type { Cluster } from '../../../entity/clusters/model/Cluster';
import { storageApi } from '../../../entity/storages/api/storageApi';
import type { Storage } from '../../../entity/storages/models/Storage';
import { IntervalType } from '../../../entity/intervals';
import { getLocalDayOfMonth, getLocalWeekday, getUserTimeFormat, getUtcDayOfMonth, getUtcWeekday } from '../../../shared/time/utils';

interface Props {
  contentHeight: number;
  workspace: WorkspaceResponse;
  isCanManageDBs: boolean;
}

export const ClustersComponent = ({ contentHeight, workspace, isCanManageDBs }: Props) => {
  const { message } = App.useApp();
  const [isLoading, setIsLoading] = useState(true);
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [storages, setStorages] = useState<Storage[]>([]);

  const [isShowAddCluster, setIsShowAddCluster] = useState(false);
  const [newCluster, setNewCluster] = useState<Cluster>({
    workspaceId: workspace.id,
    name: 'PG Cluster',
    postgresql: {
      version: '16',
      host: '',
      port: 5432,
      username: '',
      password: '',
      isHttps: false,
    },
    isBackupsEnabled: true,
    storePeriod: 'WEEK',
    storageId: undefined,
    cpuCount: 1,
    sendNotificationsOn: 'BACKUP_FAILED,BACKUP_SUCCESS',
  });

  const [selectedClusterId, setSelectedClusterId] = useState<string | null>(null);
  const [editCluster, setEditCluster] = useState<Cluster | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoadingDbs, setIsLoadingDbs] = useState(false);
  const [clusterDbs, setClusterDbs] = useState<string[]>([]);
  const [excluded, setExcluded] = useState<Set<string>>(new Set());
  const timeFmt = getUserTimeFormat();

  // Force-apply state
  const [isPropagationOpen, setIsPropagationOpen] = useState(false);
  const [propagationOpts, setPropagationOpts] = useState<ApplyPropagationRequest>({
    applyStorage: true,
    applySchedule: true,
    applyEnableBackups: true,
    respectExclusions: true,
  });
  const [previewLoading, setPreviewLoading] = useState(false);
  const [preview, setPreview] = useState<PropagationChange[]>([]);
  const [isApplyingPropagation, setIsApplyingPropagation] = useState(false);

  const loadClusters = async () => {
    setIsLoading(true);
    try {
      const list = await clusterApi.getClusters(workspace.id);
      setClusters(list);
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsLoading(false);
  };

  const loadStorages = async () => {
    try {
      const list = await storageApi.getStorages(workspace.id);
      setStorages(list);
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  useEffect(() => {
    loadClusters();
    loadStorages();
  }, [workspace.id]);

  const selectCluster = (c: Cluster) => {
    setSelectedClusterId(c.id!);
    const shallow: Cluster = JSON.parse(JSON.stringify(c));
    setEditCluster(shallow);
    const ex = new Set<string>((c.excludedDatabases || []).map((x) => x.name.toLowerCase()));
    setExcluded(ex);
    void loadClusterDatabases(c.id!);
  };

  const loadClusterDatabases = async (clusterId: string) => {
    setIsLoadingDbs(true);
    try {
      const dbs = await clusterApi.getClusterDatabases(clusterId);
      setClusterDbs(dbs);
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsLoadingDbs(false);
  };

  const createCluster = async () => {
    try {
      const created = await clusterApi.createCluster(newCluster);
      setClusters([created, ...clusters]);
      setIsShowAddCluster(false);
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const runClusterBackup = async (clusterId: string) => {
    try {
      await clusterApi.runClusterBackup(clusterId);
      message.success('Cluster backup started');
    } catch (e) {
      message.error((e as Error).message);
    }
  };

  const openPropagationModal = async () => {
    if (!selectedClusterId) return;
    setIsPropagationOpen(true);
    setPreviewLoading(true);
    try {
      const res = await clusterApi.previewPropagation(selectedClusterId, propagationOpts);
      setPreview(res);
    } catch (e) {
      message.error((e as Error).message);
    }
    setPreviewLoading(false);
  };

  const refreshPreview = async (opts: ApplyPropagationRequest) => {
    setPropagationOpts(opts);
    if (!selectedClusterId) return;
    setPreviewLoading(true);
    try {
      const res = await clusterApi.previewPropagation(selectedClusterId, opts);
      setPreview(res);
    } catch (e) {
      message.error((e as Error).message);
    }
    setPreviewLoading(false);
  };

  const applyPropagation = async () => {
    if (!selectedClusterId) return;
    setIsApplyingPropagation(true);
    try {
      const applied = await clusterApi.applyPropagation(selectedClusterId, propagationOpts);
      message.success(`Applied to ${applied.length} databases`);
      setIsPropagationOpen(false);
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsApplyingPropagation(false);
  };

  const saveCluster = async () => {
    if (!editCluster || !selectedClusterId) return;
    setIsSaving(true);
    try {
      const payload: Cluster = {
        ...editCluster,
        excludedDatabases: Array.from(excluded).map((name) => ({ name })),
      } as Cluster;
      const updated = await clusterApi.updateCluster(selectedClusterId, payload);
      // update list item
      setClusters((prev) => prev.map((c) => (c.id === updated.id ? updated : c)));
      setEditCluster(updated);
      message.success('Cluster saved');
    } catch (e) {
      message.error((e as Error).message);
    }
    setIsSaving(false);
  };

  return (
    <div className="mx-3 flex grow gap-4" style={{ height: contentHeight }}>
      <div className="w-[300px] min-w-[300px] overflow-y-auto pr-2">
        {isCanManageDBs && (
          <Button type="primary" className="mb-2 w-full" onClick={() => setIsShowAddCluster(true)}>
            Add cluster
          </Button>
        )}

      <Modal
        title="Force-apply cluster settings"
        open={isPropagationOpen}
        onCancel={() => setIsPropagationOpen(false)}
        onOk={applyPropagation}
        okText="Apply"
        confirmLoading={isApplyingPropagation}
        width={600}
      >
        <div className="mb-3">Choose what to apply to existing databases for this cluster and preview the changes before applying.</div>
        <div className="mb-3 grid grid-cols-1 gap-2 md:grid-cols-4">
          <Checkbox
            checked={propagationOpts.applyStorage}
            onChange={(e) => refreshPreview({ ...propagationOpts, applyStorage: e.target.checked })}
          >
            Apply storage
          </Checkbox>
          <Checkbox
            checked={propagationOpts.applySchedule}
            onChange={(e) => refreshPreview({ ...propagationOpts, applySchedule: e.target.checked })}
          >
            Apply schedule
          </Checkbox>
          <Checkbox
            checked={propagationOpts.applyEnableBackups}
            onChange={(e) => refreshPreview({ ...propagationOpts, applyEnableBackups: e.target.checked })}
          >
            Apply enable backups
          </Checkbox>
          <Checkbox
            checked={propagationOpts.respectExclusions}
            onChange={(e) => refreshPreview({ ...propagationOpts, respectExclusions: e.target.checked })}
          >
            Respect exclusions
          </Checkbox>
        </div>

        {previewLoading ? (
          <div className="py-3"><Spin /></div>
        ) : preview.length === 0 ? (
          <div className="text-xs text-gray-500">No databases would be changed with the current options.</div>
        ) : (
          <div className="max-h-64 overflow-y-auto rounded border">
            {preview.map((p) => (
              <div key={p.databaseId} className="flex items-center justify-between border-b px-2 py-1 text-sm last:border-b-0">
                <div className="truncate pr-2">{p.name}</div>
                <div className="text-xs text-gray-600">
                  {p.changeStorage ? 'Storage' : ''}{p.changeStorage && p.changeSchedule ? ' · ' : ''}{p.changeSchedule ? 'Schedule' : ''}
                </div>
              </div>
            ))}
          </div>
        )}
      </Modal>

        <div className="mb-2 flex items-center justify-between">
          <div className="text-xs text-gray-500">{clusters.length} clusters</div>
          <Button size="small" onClick={() => loadClusters()}>
            Refresh
          </Button>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-3">
            <Spin />
          </div>
        ) : clusters.length === 0 ? (
          <div className="mx-3 text-center text-xs text-gray-500">No clusters yet</div>
        ) : (
          clusters.map((c) => (
            <div
              key={c.id}
              className={`mb-2 cursor-pointer rounded border p-2 ${selectedClusterId === c.id ? 'border-blue-400 bg-blue-50' : 'border-gray-200'}`}
              onClick={() => selectCluster(c)}
            >
              <div className="font-medium">{c.name}</div>
              <div className="text-xs text-gray-500">
                {c.postgresql.host}:{c.postgresql.port} · {c.postgresql.username}
              </div>
              <div className="mt-2 flex gap-2">
                <Button size="small" onClick={() => selectCluster(c)}>
                  Manage
                </Button>
                <Button size="small" type="primary" onClick={() => runClusterBackup(c.id!)}>
                  Run backup
                </Button>
              </div>
            </div>
          ))
        )}
      </div>

      <div className="grow overflow-y-auto rounded bg-white p-4 shadow">
        {!editCluster ? (
          <div className="text-gray-600">Select a cluster on the left to manage settings, storage, and exclusions.</div>
        ) : (
          <div className="max-w-3xl">
            <div className="mb-4 text-lg font-semibold">Cluster settings</div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Name</div>
              <Input
                value={editCluster.name}
                onChange={(e) => setEditCluster({ ...(editCluster as Cluster), name: e.target.value })}
                size="small"
                className="max-w-[280px] grow"
                placeholder="Cluster name"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">PG version</div>
              <Select
                value={editCluster.postgresql.version}
                onChange={(v) => setEditCluster({ ...(editCluster as Cluster), postgresql: { ...(editCluster!.postgresql), version: v as string } })}
                size="small"
                className="max-w-[280px] grow"
                options={[
                  { label: '12', value: '12' },
                  { label: '13', value: '13' },
                  { label: '14', value: '14' },
                  { label: '15', value: '15' },
                  { label: '16', value: '16' },
                  { label: '17', value: '17' },
                  { label: '18', value: '18' },
                ]}
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Host</div>
              <Input
                value={editCluster.postgresql.host}
                onChange={(e) => setEditCluster({ ...(editCluster as Cluster), postgresql: { ...(editCluster!.postgresql), host: e.target.value.trim().replace('https://', '').replace('http://', '') } })}
                size="small"
                className="max-w-[280px] grow"
                placeholder="Enter PG host"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Port</div>
              <InputNumber
                type="number"
                value={editCluster.postgresql.port}
                onChange={(v) => typeof v === 'number' && setEditCluster({ ...(editCluster as Cluster), postgresql: { ...(editCluster!.postgresql), port: v } })}
                size="small"
                className="max-w-[280px] grow"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Username</div>
              <Input
                value={editCluster.postgresql.username}
                onChange={(e) => setEditCluster({ ...(editCluster as Cluster), postgresql: { ...(editCluster!.postgresql), username: e.target.value.trim() } })}
                size="small"
                className="max-w-[280px] grow"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Password</div>
              <Input.Password
                value={''}
                onChange={(e) => setEditCluster({ ...(editCluster as Cluster), postgresql: { ...(editCluster!.postgresql), password: e.target.value } })}
                size="small"
                className="max-w-[280px] grow"
                placeholder="Leave empty to keep"
              />
            </div>

            <div className="mb-3 flex w-full items-center">
              <div className="min-w-[160px]">Use HTTPS</div>
              <Switch
                checked={!!editCluster.postgresql.isHttps}
                onChange={(checked) => setEditCluster({ ...(editCluster as Cluster), postgresql: { ...(editCluster!.postgresql), isHttps: checked } })}
                size="small"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Enable backups</div>
              <Switch
                checked={!!editCluster.isBackupsEnabled}
                onChange={(checked) => setEditCluster({ ...(editCluster as Cluster), isBackupsEnabled: checked })}
                size="small"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Store period</div>
              <Select
                value={editCluster.storePeriod}
                onChange={(v) => setEditCluster({ ...(editCluster as Cluster), storePeriod: v as string })}
                size="small"
                className="max-w-[280px] grow"
                options={[
                  { label: 'Day', value: 'DAY' },
                  { label: 'Week', value: 'WEEK' },
                  { label: 'Month', value: 'MONTH' },
                  { label: '3 months', value: '3_MONTH' },
                  { label: '6 months', value: '6_MONTH' },
                  { label: 'Year', value: 'YEAR' },
                  { label: '2 years', value: '2_YEARS' },
                  { label: '3 years', value: '3_YEARS' },
                  { label: '4 years', value: '4_YEARS' },
                  { label: '5 years', value: '5_YEARS' },
                  { label: 'Forever', value: 'FOREVER' },
                ]}
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">CPU count</div>
              <InputNumber
                type="number"
                min={1}
                max={8}
                value={editCluster.cpuCount || 1}
                onChange={(v) => typeof v === 'number' && setEditCluster({ ...(editCluster as Cluster), cpuCount: v })}
                size="small"
                className="max-w-[280px] grow"
              />
            </div>

            <div className="mb-2 flex w-full items-center">
              <div className="min-w-[160px]">Storage</div>
              <Select
                value={editCluster.storageId}
                onChange={(v) => setEditCluster({ ...(editCluster as Cluster), storageId: v as string })}
                size="small"
                className="max-w-[280px] grow"
                allowClear
                placeholder="Select storage"
                options={storages.map((s) => ({ label: s.name, value: s.id }))}
              />
            </div>

            {editCluster.isBackupsEnabled && (
              <>
                <div className="mt-3 mb-1 flex w-full items-center">
                  <div className="min-w-[160px]">Backup interval</div>
                  <Select
                    value={editCluster.backupInterval?.interval || IntervalType.DAILY}
                    onChange={(v) => setEditCluster({ ...(editCluster as Cluster), backupInterval: { ...(editCluster.backupInterval || { id: undefined as unknown as string }), interval: v as IntervalType, timeOfDay: editCluster.backupInterval?.timeOfDay || '04:00' } })}
                    size="small"
                    className="max-w-[280px] grow"
                    options={[
                      { label: 'Hourly', value: IntervalType.HOURLY },
                      { label: 'Daily', value: IntervalType.DAILY },
                      { label: 'Weekly', value: IntervalType.WEEKLY },
                      { label: 'Monthly', value: IntervalType.MONTHLY },
                    ]}
                  />
                </div>

                {editCluster.backupInterval?.interval === IntervalType.WEEKLY && (
                  <div className="mb-1 flex w-full items-center">
                    <div className="min-w-[160px]">Backup weekday</div>
                    <Select
                      value={editCluster.backupInterval?.weekday ? getLocalWeekday(editCluster.backupInterval!.weekday!, editCluster.backupInterval!.timeOfDay) : undefined}
                      onChange={(localW: number) => {
                        const ref = dayjs(editCluster.backupInterval?.timeOfDay || '04:00', 'HH:mm');
                        setEditCluster({
                          ...(editCluster as Cluster),
                          backupInterval: {
                            ...(editCluster.backupInterval || { id: undefined as unknown as string, interval: IntervalType.WEEKLY, timeOfDay: '04:00' }),
                            weekday: getUtcWeekday(localW, ref),
                          },
                        });
                      }}
                      size="small"
                      className="max-w-[280px] grow"
                      options={[
                        { value: 1, label: 'Mon' },
                        { value: 2, label: 'Tue' },
                        { value: 3, label: 'Wed' },
                        { value: 4, label: 'Thu' },
                        { value: 5, label: 'Fri' },
                        { value: 6, label: 'Sat' },
                        { value: 7, label: 'Sun' },
                      ]}
                    />
                  </div>
                )}

                {editCluster.backupInterval?.interval === IntervalType.MONTHLY && (
                  <div className="mb-1 flex w-full items-center">
                    <div className="min-w-[160px]">Backup day of month</div>
                    <InputNumber
                      min={1}
                      max={31}
                      value={editCluster.backupInterval?.dayOfMonth ? getLocalDayOfMonth(editCluster.backupInterval!.dayOfMonth!, editCluster.backupInterval!.timeOfDay) : undefined}
                      onChange={(localDom) => {
                        if (!localDom) return;
                        const ref = dayjs(editCluster.backupInterval?.timeOfDay || '04:00', 'HH:mm');
                        setEditCluster({
                          ...(editCluster as Cluster),
                          backupInterval: {
                            ...(editCluster.backupInterval || { id: undefined as unknown as string, interval: IntervalType.MONTHLY, timeOfDay: '04:00' }),
                            dayOfMonth: getUtcDayOfMonth(localDom, ref),
                          },
                        });
                      }}
                      size="small"
                      className="max-w-[280px] grow"
                    />
                  </div>
                )}

                {editCluster.backupInterval?.interval !== IntervalType.HOURLY && (
                  <div className="mb-1 flex w-full items-center">
                    <div className="min-w-[160px]">Backup time of day</div>
                    <TimePicker
                      value={dayjs(editCluster.backupInterval?.timeOfDay || '04:00', 'HH:mm')}
                      use12Hours={timeFmt}
                      format={timeFmt ? 'h:mm A' : 'HH:mm'}
                      allowClear={false}
                      size="small"
                      className="max-w-[280px] grow"
                      onChange={(t: Dayjs | null) => {
                        if (!t) return;
                        setEditCluster({
                          ...(editCluster as Cluster),
                          backupInterval: {
                            ...(editCluster.backupInterval || { id: undefined as unknown as string, interval: IntervalType.DAILY }),
                            timeOfDay: t.utc().format('HH:mm'),
                          },
                        });
                      }}
                    />
                  </div>
                )}
              </>
            )}

            <div className="mt-4 flex">
              <Button className="mr-2" onClick={() => selectedClusterId && runClusterBackup(selectedClusterId)}>
                Run backup
              </Button>
              <Button className="mr-auto" onClick={() => selectedClusterId && loadClusterDatabases(selectedClusterId)} disabled={!selectedClusterId}>
                Reload databases
              </Button>
              <Button className="mr-2" onClick={openPropagationModal} disabled={!selectedClusterId}>
                Force-apply to DBs
              </Button>
              <Button type="primary" loading={isSaving} onClick={saveCluster}>
                Save
              </Button>
            </div>

            <div className="mt-6 border-t pt-4">
              <div className="mb-2 text-base font-semibold">Databases in cluster</div>
              {isLoadingDbs ? (
                <div className="py-3">
                  <Spin />
                </div>
              ) : clusterDbs.length === 0 ? (
                <div className="text-xs text-gray-500">No databases found or cannot connect</div>
              ) : (
                <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
                  {clusterDbs.map((name) => (
                    <div key={name} className="flex items-center justify-between rounded border border-gray-200 px-2 py-1 text-sm">
                      <div className="truncate pr-2">{name}</div>
                      <Checkbox
                        checked={excluded.has(name.toLowerCase())}
                        onChange={(e) => {
                          const next = new Set(excluded);
                          if (e.target.checked) next.add(name.toLowerCase());
                          else next.delete(name.toLowerCase());
                          setExcluded(next);
                        }}
                      >
                        Exclude
                      </Checkbox>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {isShowAddCluster && (
        <Modal
          title="Add cluster"
          footer={<div />}
          open={isShowAddCluster}
          onCancel={() => setIsShowAddCluster(false)}
          width={520}
        >
          <div className="mt-2" />
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Name</div>
            <Input
              value={newCluster.name}
              onChange={(e) => setNewCluster({ ...newCluster, name: e.target.value })}
              size="small"
              className="max-w-[220px] grow"
              placeholder="Cluster name"
            />
          </div>

          <div className="mb-1 flex w/full items-center">
            <div className="min-w-[150px]">PG version</div>
            <Select
              value={newCluster.postgresql.version}
              onChange={(v) => setNewCluster({ ...newCluster, postgresql: { ...newCluster.postgresql, version: v as string } })}
              size="small"
              className="max-w-[220px] grow"
              placeholder="Select PG version"
              options={[
                { label: '12', value: '12' },
                { label: '13', value: '13' },
                { label: '14', value: '14' },
                { label: '15', value: '15' },
                { label: '16', value: '16' },
                { label: '17', value: '17' },
                { label: '18', value: '18' },
              ]}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w/[150px]">Host</div>
            <Input
              value={newCluster.postgresql.host}
              onChange={(e) => setNewCluster({ ...newCluster, postgresql: { ...newCluster.postgresql, host: e.target.value.trim().replace('https://', '').replace('http://', '') } })}
              size="small"
              className="max-w-[220px] grow"
              placeholder="Enter PG host"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w/[150px]">Port</div>
            <InputNumber
              type="number"
              value={newCluster.postgresql.port}
              onChange={(v) => typeof v === 'number' && setNewCluster({ ...newCluster, postgresql: { ...newCluster.postgresql, port: v } })}
              size="small"
              className="max-w-[220px] grow"
              placeholder="Enter PG port"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w/[150px]">Username</div>
            <Input
              value={newCluster.postgresql.username}
              onChange={(e) => setNewCluster({ ...newCluster, postgresql: { ...newCluster.postgresql, username: e.target.value.trim() } })}
              size="small"
              className="max-w/[220px] grow"
              placeholder="Enter PG username"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w/[150px]">Password</div>
            <Input.Password
              value={newCluster.postgresql.password}
              onChange={(e) => setNewCluster({ ...newCluster, postgresql: { ...newCluster.postgresql, password: e.target.value.trim() } })}
              size="small"
              className="max-w/[220px] grow"
              placeholder="Enter PG password"
            />
          </div>

          <div className="mb-3 flex w-full items-center">
            <div className="min-w/[150px]">Use HTTPS</div>
            <Switch
              checked={newCluster.postgresql.isHttps}
              onChange={(checked) => setNewCluster({ ...newCluster, postgresql: { ...newCluster.postgresql, isHttps: checked } })}
              size="small"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Enable backups</div>
            <Switch
              checked={!!newCluster.isBackupsEnabled}
              onChange={(checked) => setNewCluster({ ...newCluster, isBackupsEnabled: checked })}
              size="small"
            />
          </div>

          {newCluster.isBackupsEnabled && (
            <>
              <div className="mb-1 flex w-full items-center">
                <div className="min-w-[150px]">Backup interval</div>
                <Select
                  value={newCluster.backupInterval?.interval || IntervalType.DAILY}
                  onChange={(v) => setNewCluster({ ...newCluster, backupInterval: { ...(newCluster.backupInterval || { id: undefined as unknown as string }), interval: v as IntervalType, timeOfDay: newCluster.backupInterval?.timeOfDay || '04:00' } })}
                  size="small"
                  className="max-w-[220px] grow"
                  options={[
                    { label: 'Hourly', value: IntervalType.HOURLY },
                    { label: 'Daily', value: IntervalType.DAILY },
                    { label: 'Weekly', value: IntervalType.WEEKLY },
                    { label: 'Monthly', value: IntervalType.MONTHLY },
                  ]}
                />
              </div>

              {newCluster.backupInterval?.interval === IntervalType.WEEKLY && (
                <div className="mb-1 flex w-full items-center">
                  <div className="min-w-[150px]">Backup weekday</div>
                  <Select
                    value={newCluster.backupInterval?.weekday}
                    onChange={(wd) => setNewCluster({ ...newCluster, backupInterval: { ...(newCluster.backupInterval || { id: undefined as unknown as string, interval: IntervalType.WEEKLY, timeOfDay: '04:00' }), weekday: wd as number } })}
                    size="small"
                    className="max-w-[220px] grow"
                    options={[
                      { value: 1, label: 'Mon' },
                      { value: 2, label: 'Tue' },
                      { value: 3, label: 'Wed' },
                      { value: 4, label: 'Thu' },
                      { value: 5, label: 'Fri' },
                      { value: 6, label: 'Sat' },
                      { value: 7, label: 'Sun' },
                    ]}
                  />
                </div>
              )}

              {newCluster.backupInterval?.interval === IntervalType.MONTHLY && (
                <div className="mb-1 flex w-full items-center">
                  <div className="min-w-[150px]">Backup day of month</div>
                  <InputNumber
                    min={1}
                    max={31}
                    value={newCluster.backupInterval?.dayOfMonth}
                    onChange={(v) => typeof v === 'number' && setNewCluster({ ...newCluster, backupInterval: { ...(newCluster.backupInterval || { id: undefined as unknown as string, interval: IntervalType.MONTHLY, timeOfDay: '04:00' }), dayOfMonth: v } })}
                    size="small"
                    className="max-w-[220px] grow"
                  />
                </div>
              )}

              {newCluster.backupInterval?.interval !== IntervalType.HOURLY && (
                <div className="mb-1 flex w-full items-center">
                  <div className="min-w-[150px]">Backup time of day</div>
                  <TimePicker
                    value={dayjs(newCluster.backupInterval?.timeOfDay || '04:00', 'HH:mm')}
                    use12Hours={timeFmt}
                    format={timeFmt ? 'h:mm A' : 'HH:mm'}
                    allowClear={false}
                    size="small"
                    className="max-w-[220px] grow"
                    onChange={(t) => t && setNewCluster({ ...newCluster, backupInterval: { ...(newCluster.backupInterval || { id: undefined as unknown as string, interval: IntervalType.DAILY }), timeOfDay: t.utc().format('HH:mm') } })}
                  />
                </div>
              )}
            </>
          )}

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">CPU count</div>
            <InputNumber
              type="number"
              min={1}
              max={8}
              value={newCluster.cpuCount}
              onChange={(v) => typeof v === 'number' && setNewCluster({ ...newCluster, cpuCount: v })}
              size="small"
              className="max-w/[220px] grow"
              placeholder="CPU count for pg_dump"
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Store period</div>
            <Select
              value={newCluster.storePeriod}
              onChange={(v) => setNewCluster({ ...newCluster, storePeriod: v as string })}
              size="small"
              className="max-w-[220px] grow"
              options={[
                { label: 'Day', value: 'DAY' },
                { label: 'Week', value: 'WEEK' },
                { label: 'Month', value: 'MONTH' },
                { label: '3 months', value: '3_MONTH' },
                { label: '6 months', value: '6_MONTH' },
                { label: 'Year', value: 'YEAR' },
                { label: '2 years', value: '2_YEARS' },
                { label: '3 years', value: '3_YEARS' },
                { label: '4 years', value: '4_YEARS' },
                { label: '5 years', value: '5_YEARS' },
                { label: 'Forever', value: 'FOREVER' },
              ]}
            />
          </div>

          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">CPU count</div>
            <InputNumber
              type="number"
              min={1}
              max={8}
              value={newCluster.cpuCount}
              onChange={(v) => typeof v === 'number' && setNewCluster({ ...newCluster, cpuCount: v })}
              size="small"
              className="max-w/[220px] grow"
              placeholder="CPU count for pg_dump"
            />
          </div>

          <div className="mb-3 flex w-full items-center">
            <div className="min-w-[150px]">Storage</div>
            <Select
              value={newCluster.storageId}
              onChange={(v) => setNewCluster({ ...newCluster, storageId: v as string })}
              size="small"
              className="max-w-[220px] grow"
              allowClear
              placeholder="Select storage"
              options={storages.map((s) => ({ label: s.name, value: s.id }))}
            />
          </div>

          <div className="mt-4 flex">
            <Button className="mr-auto" onClick={() => setIsShowAddCluster(false)}>
              Cancel
            </Button>
            <Button
              type="primary"
              className="ml-1"
              onClick={createCluster}
              disabled={
                !newCluster.name ||
                !newCluster.postgresql.version ||
                !newCluster.postgresql.host ||
                !newCluster.postgresql.port ||
                !newCluster.postgresql.username ||
                !newCluster.postgresql.password
              }
            >
              Create
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
};

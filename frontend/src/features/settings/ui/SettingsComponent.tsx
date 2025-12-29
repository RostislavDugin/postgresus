import { LoadingOutlined } from '@ant-design/icons';
import { App, Button, Spin, Switch } from 'antd';
import { useEffect, useRef, useState } from 'react';

import { IS_CLOUD, getApplicationServer } from '../../../constants';
import { settingsApi } from '../../../entity/users/api/settingsApi';
import type { UsersSettings } from '../../../entity/users/model/UsersSettings';
import { AuditLogsComponent } from './AuditLogsComponent';

interface Props {
  contentHeight: number;
}

export function SettingsComponent({ contentHeight }: Props) {
  const { message } = App.useApp();
  const [settings, setSettings] = useState<UsersSettings | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);

  // Scroll container ref for audit logs lazy loading
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Form state to track changes
  const [formSettings, setFormSettings] = useState<UsersSettings>({
    isAllowExternalRegistrations: false,
    isAllowMemberInvitations: false,
    isMemberAllowedToCreateWorkspaces: false,
  });

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    setIsLoading(true);

    try {
      const currentSettings = await settingsApi.getSettings();
      setSettings(currentSettings);
      setFormSettings(currentSettings);
      setHasChanges(false);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load settings';
      message.error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSettingChange = (key: keyof UsersSettings, value: boolean) => {
    const newFormSettings = { ...formSettings, [key]: value };
    setFormSettings(newFormSettings);

    // Check if there are changes from the original settings
    if (settings) {
      const hasAnyChanges = Object.keys(newFormSettings).some(
        (settingKey) =>
          newFormSettings[settingKey as keyof UsersSettings] !==
          settings[settingKey as keyof UsersSettings],
      );
      setHasChanges(hasAnyChanges);
    }
  };

  const handleSave = async () => {
    if (!hasChanges) return;

    setIsSaving(true);
    try {
      const updatedSettings = await settingsApi.updateSettings(formSettings);
      setSettings(updatedSettings);
      setFormSettings(updatedSettings);
      setHasChanges(false);
      message.success('Settings updated successfully');
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to update settings';
      message.error(errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const handleReset = () => {
    if (settings) {
      setFormSettings(settings);
      setHasChanges(false);
    }
  };

  console.log(`isCloud = ${IS_CLOUD}`);

  return (
    <div className="flex grow sm:pl-5">
      <div className="w-full">
        <div
          ref={scrollContainerRef}
          className="grow overflow-y-auto rounded bg-white p-5 shadow dark:bg-gray-800"
          style={{ height: contentHeight }}
        >
          <h1 className="text-2xl font-bold dark:text-white">Databasus settings</h1>

          <div className="mt-6">
            {isLoading ? (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            ) : (
              <div className="max-w-lg text-sm">
                <div className="space-y-6">
                  {/* External Registrations Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900 dark:text-white">
                        Allow external registrations
                      </div>
                      <div className="mt-1 text-gray-500 dark:text-gray-400">
                        When enabled, new users can register accounts in Databasus. If disabled, new
                        users can only register via invitation
                      </div>
                    </div>

                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isAllowExternalRegistrations}
                        onChange={(checked) =>
                          handleSettingChange('isAllowExternalRegistrations', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isAllowExternalRegistrations
                            ? '#155dfc'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>

                  {/* Member Invitations Setting */}
                  {!formSettings.isAllowExternalRegistrations && (
                    <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                      <div className="flex-1 pr-20">
                        <div className="font-medium text-gray-900 dark:text-white">
                          Allow member invitations
                        </div>

                        <div className="mt-1 text-gray-500 dark:text-gray-400">
                          When enabled, existing members can invite new users to join Databasus. If
                          not - only admins can invite users.
                        </div>
                      </div>

                      <div className="ml-4">
                        <Switch
                          checked={formSettings.isAllowMemberInvitations}
                          onChange={(checked) =>
                            handleSettingChange('isAllowMemberInvitations', checked)
                          }
                          style={{
                            backgroundColor: formSettings.isAllowMemberInvitations
                              ? '#155dfc'
                              : undefined,
                          }}
                        />
                      </div>
                    </div>
                  )}

                  {/* Member Workspace Creation Setting */}
                  <div className="flex items-start justify-between border-b border-gray-200 pb-4 dark:border-gray-700">
                    <div className="flex-1 pr-20">
                      <div className="font-medium text-gray-900 dark:text-white">
                        Members can create workspaces
                      </div>

                      <div className="mt-1 text-gray-500 dark:text-gray-400">
                        When enabled, members (non-admin users) can create new workspaces. If not -
                        only admins can create workspaces.
                      </div>
                    </div>
                    <div className="ml-4">
                      <Switch
                        checked={formSettings.isMemberAllowedToCreateWorkspaces}
                        onChange={(checked) =>
                          handleSettingChange('isMemberAllowedToCreateWorkspaces', checked)
                        }
                        style={{
                          backgroundColor: formSettings.isMemberAllowedToCreateWorkspaces
                            ? '#155dfc'
                            : undefined,
                        }}
                      />
                    </div>
                  </div>
                </div>

                {/* Action Buttons */}
                {hasChanges && (
                  <div className="mt-8 flex space-x-2">
                    <Button
                      type="primary"
                      onClick={handleSave}
                      loading={isSaving}
                      disabled={isSaving}
                      className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                    >
                      {isSaving ? 'Saving...' : 'Save Changes'}
                    </Button>

                    <Button type="default" onClick={handleReset} disabled={isSaving}>
                      Reset
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
            Read more about settings you can{' '}
            <a
              href="https://databasus.com/access-management/#global-settings"
              target="_blank"
              rel="noreferrer"
              className="!text-blue-600"
            >
              here
            </a>
          </div>

          {/* Health-check Information */}
          <div className="my-8 max-w-2xl">
            <h2 className="mb-3 text-xl font-bold dark:text-white">Health-check</h2>

            <div className="group relative">
              <div className="flex items-center rounded-md border border-gray-300 bg-gray-50 px-3 py-2 !font-mono text-sm text-gray-700 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200">
                <code
                  className="flex-1 cursor-pointer break-all transition-colors select-all hover:text-blue-600"
                  onClick={() => {
                    window.open(`${getApplicationServer()}/api/v1/system/health`, '_blank');
                  }}
                  title="Click to open in new tab"
                >
                  {getApplicationServer()}/api/v1/system/health
                </code>
                <Button
                  type="text"
                  size="small"
                  className="ml-2 opacity-0 transition-opacity group-hover:opacity-100"
                  onClick={() => {
                    navigator.clipboard.writeText(`${getApplicationServer()}/api/v1/system/health`);
                    message.success('Health-check endpoint copied to clipboard');
                  }}
                >
                  📋
                </Button>
              </div>
              <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Use this endpoint to monitor your Databasus system&apos;s availability
              </div>
            </div>
          </div>

          <AuditLogsComponent scrollContainerRef={scrollContainerRef} />
        </div>
      </div>
    </div>
  );
}

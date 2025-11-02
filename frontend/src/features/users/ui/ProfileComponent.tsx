import { EyeInvisibleOutlined, EyeTwoTone, LoadingOutlined } from '@ant-design/icons';
import { App, Button, Input, Spin } from 'antd';
import { useEffect, useState } from 'react';

import { userApi } from '../../../entity/users/api/userApi';
import type { ChangePasswordRequest } from '../../../entity/users/model/ChangePasswordRequest';
import type { SignInRequest } from '../../../entity/users/model/SignInRequest';
import type { UpdateUserInfoRequest } from '../../../entity/users/model/UpdateUserInfoRequest';
import type { UserProfile } from '../../../entity/users/model/UserProfile';
import { UserRole } from '../../../entity/users/model/UserRole';

interface Props {
  contentHeight: number;
}

const getRoleDisplayText = (role: UserRole): string => {
  switch (role) {
    case UserRole.ADMIN:
      return 'Admin';
    case UserRole.MEMBER:
      return 'Member';
    default:
      return role;
  }
};

export function ProfileComponent({ contentHeight }: Props) {
  const { message } = App.useApp();
  const [user, setUser] = useState<UserProfile | undefined>(undefined);
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  // Profile edit state
  const [editName, setEditName] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false);
  const [editNameError, setEditNameError] = useState(false);
  const [editEmailError, setEditEmailError] = useState(false);

  // Password change form state
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [newPasswordVisible, setNewPasswordVisible] = useState(false);
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  // Error states
  const [newPasswordError, setNewPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  useEffect(() => {
    loadUserProfile();
  }, []);

  const loadUserProfile = () => {
    userApi
      .getCurrentUser()
      .then((user) => {
        setUser(user);
        setEditName(user.name);
        setEditEmail(user.email);
      })
      .catch((error) => {
        message.error(error.message);
      });
  };

  const validatePasswordFields = (): boolean => {
    let isValid = true;

    if (!newPassword) {
      setNewPasswordError(true);
      isValid = false;
    } else if (newPassword.length < 6) {
      setNewPasswordError(true);
      message.error('Password must be at least 6 characters long');
      isValid = false;
    } else {
      setNewPasswordError(false);
    }

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      isValid = false;
    } else if (newPassword !== confirmPassword) {
      setConfirmPasswordError(true);
      message.error('New passwords do not match');
      isValid = false;
    } else {
      setConfirmPasswordError(false);
    }

    return isValid;
  };

  const handlePasswordChange = async () => {
    if (!validatePasswordFields()) {
      return;
    }

    setIsChangingPassword(true);

    try {
      const request: ChangePasswordRequest = {
        newPassword,
      };

      await userApi.changePassword(request);

      // Reset form fields
      setNewPassword('');
      setConfirmPassword('');

      // Sign in again with new password
      if (user?.email) {
        try {
          const signInRequest: SignInRequest = {
            email: user.email,
            password: newPassword,
          };
          await userApi.signIn(signInRequest);
          message.success('Successfully signed in with new password');
        } catch (signInError: unknown) {
          const errorMessage =
            signInError instanceof Error
              ? signInError.message
              : 'Failed to sign in with new password';
          message.error(errorMessage);
          // If sign in fails, logout and redirect to login page
          userApi.logout();
          userApi.notifyAuthListeners();
          window.location.reload();
        }
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to change password';
      message.error(errorMessage);
    } finally {
      setIsChangingPassword(false);
    }
  };

  const handleProfileUpdate = async () => {
    // Validate name
    if (!editName || editName.trim() === '') {
      setEditNameError(true);
      message.error('Name is required');
      return;
    }
    setEditNameError(false);

    // Validate email (only if not admin)
    if (user?.email !== 'admin') {
      if (!editEmail || editEmail.trim() === '') {
        setEditEmailError(true);
        message.error('Email is required');
        return;
      }
      setEditEmailError(false);
    }

    setIsUpdatingProfile(true);

    try {
      const request: UpdateUserInfoRequest = {};

      // Only include fields that changed
      if (editName !== user?.name) {
        request.name = editName;
      }
      // Only include email if not admin and changed
      if (user?.email !== 'admin' && editEmail !== user?.email) {
        request.email = editEmail;
      }

      // If nothing changed, just show a message
      if (Object.keys(request).length === 0) {
        message.info('No changes to save');
        setIsUpdatingProfile(false);
        return;
      }

      await userApi.updateUserInfo(request);
      message.success('Profile updated successfully');

      // Reload user profile
      loadUserProfile();
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to update profile';
      message.error(errorMessage);
    } finally {
      setIsUpdatingProfile(false);
    }
  };

  const handleLogout = () => {
    userApi.logout();
    window.location.reload();
  };

  return (
    <div className="flex grow pl-3">
      <div className="w-full">
        <div
          className="grow overflow-y-auto rounded bg-white p-5 shadow"
          style={{ height: contentHeight }}
        >
          <h1 className="text-2xl font-bold">Profile</h1>

          <div className="mt-5">
            {user ? (
              <>
                <div className="mb-6">
                  <h3 className="mb-4 text-lg font-semibold">Profile Information</h3>
                  <div className="max-w-md">
                    <div className="text-xs font-semibold">User ID</div>
                    <div className="mb-4 text-sm text-gray-600">{user.id}</div>

                    <div className="mb-1 text-xs font-semibold">Name</div>
                    <Input
                      value={editName}
                      onChange={(e) => {
                        setEditNameError(false);
                        setEditName(e.currentTarget.value);
                      }}
                      status={editNameError ? 'error' : undefined}
                      placeholder="Enter your name"
                      className="mb-4"
                    />

                    <div className="mt-2 mb-1 text-xs font-semibold">Email</div>
                    <Input
                      value={editEmail}
                      onChange={(e) => {
                        setEditEmailError(false);
                        setEditEmail(e.currentTarget.value.trim().toLowerCase());
                      }}
                      status={editEmailError ? 'error' : undefined}
                      placeholder="Enter your email"
                      type="email"
                      className="mb-4"
                      disabled={user.email === 'admin'}
                    />
                    {user.email === 'admin' && (
                      <div className="mb-4 text-xs text-gray-500">
                        Admin email cannot be changed
                      </div>
                    )}

                    <div className="mt-2 mb-1 text-xs font-semibold">Role</div>
                    <div className="mb-4">
                      <span className="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800">
                        {getRoleDisplayText(user.role)}
                      </span>
                    </div>

                    {(editName !== user.name || editEmail !== user.email) && (
                      <Button
                        type="primary"
                        onClick={handleProfileUpdate}
                        loading={isUpdatingProfile}
                        disabled={isUpdatingProfile}
                        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                      >
                        Save changes
                      </Button>
                    )}
                  </div>
                </div>

                <div className="mb-8">
                  <Button type="default" onClick={handleLogout} danger>
                    Logout
                  </Button>
                </div>

                <div className="max-w-xs">
                  <h3 className="mb-4 text-lg font-semibold">Change Password</h3>

                  <div className="max-w-sm">
                    <div className="my-1 text-xs font-semibold">New Password</div>
                    <Input.Password
                      placeholder="Enter new password"
                      value={newPassword}
                      onChange={(e) => {
                        setNewPasswordError(false);
                        setNewPassword(e.currentTarget.value);
                      }}
                      status={newPasswordError ? 'error' : undefined}
                      iconRender={(visible) =>
                        visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                      }
                      visibilityToggle={{
                        visible: newPasswordVisible,
                        onVisibleChange: setNewPasswordVisible,
                      }}
                    />

                    <div className="mt-2 mb-1 text-xs font-semibold">Confirm New Password</div>
                    <Input.Password
                      placeholder="Confirm new password"
                      value={confirmPassword}
                      onChange={(e) => {
                        setConfirmPasswordError(false);
                        setConfirmPassword(e.currentTarget.value);
                      }}
                      status={confirmPasswordError ? 'error' : undefined}
                      iconRender={(visible) =>
                        visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />
                      }
                      visibilityToggle={{
                        visible: confirmPasswordVisible,
                        onVisibleChange: setConfirmPasswordVisible,
                      }}
                    />

                    <div className="mt-3" />

                    {(newPassword || confirmPassword) && (
                      <Button
                        type="primary"
                        onClick={handlePasswordChange}
                        loading={isChangingPassword}
                        disabled={isChangingPassword}
                        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
                      >
                        {isChangingPassword ? 'Changing password...' : 'Change password'}
                      </Button>
                    )}
                  </div>
                </div>
              </>
            ) : (
              <div>
                <Spin indicator={<LoadingOutlined spin />} />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

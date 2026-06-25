import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import { useEffect, useState } from 'react';

import { userApi } from '../entity/users';
import {
  AdminPasswordComponent,
  AuthNavbarComponent,
  RequestResetPasswordComponent,
  ResetPasswordComponent,
  SignInComponent,
  SignUpComponent,
} from '../features/users';
import { useScreenHeight } from '../shared/hooks';

export function AuthPageComponent() {
  const [isAdminHasPassword, setIsAdminHasPassword] = useState(false);
  const [authMode, setAuthMode] = useState<'signIn' | 'signUp' | 'requestReset' | 'resetPassword'>(
    'signUp',
  );
  const [resetEmail, setResetEmail] = useState('');
  const [isLoading, setLoading] = useState(true);
  const screenHeight = useScreenHeight();

  const checkAdminPasswordStatus = () => {
    setLoading(true);

    userApi
      .isAdminHasPassword()
      .then((response) => {
        setIsAdminHasPassword(response.hasPassword);
        setLoading(false);
      })
      .catch((e) => {
        alert('Failed to check admin password status: ' + (e as Error).message);
      });
  };

  useEffect(() => {
    checkAdminPasswordStatus();
  }, []);

  return (
    <div className="flex min-h-full flex-col dark:bg-gray-900" style={{ minHeight: screenHeight }}>
      {isLoading ? (
        <div className="flex h-screen w-screen items-center justify-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        </div>
      ) : (
        <div>
          <div>
            <AuthNavbarComponent />

            <div className="mt-10 flex justify-center sm:mt-[10vh]">
              {isAdminHasPassword ? (
                authMode === 'signUp' ? (
                  <SignUpComponent onSwitchToSignIn={() => setAuthMode('signIn')} />
                ) : authMode === 'signIn' ? (
                  <SignInComponent
                    onSwitchToSignUp={() => setAuthMode('signUp')}
                    onSwitchToResetPassword={() => setAuthMode('requestReset')}
                  />
                ) : authMode === 'requestReset' ? (
                  <RequestResetPasswordComponent
                    onSwitchToSignIn={() => setAuthMode('signIn')}
                    onSwitchToResetPassword={(email) => {
                      setResetEmail(email);
                      setAuthMode('resetPassword');
                    }}
                  />
                ) : (
                  <ResetPasswordComponent
                    onSwitchToSignIn={() => setAuthMode('signIn')}
                    onSwitchToRequestCode={() => setAuthMode('requestReset')}
                    initialEmail={resetEmail}
                  />
                )
              ) : (
                <AdminPasswordComponent onPasswordSet={checkAdminPasswordStatus} />
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

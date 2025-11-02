import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import { useEffect, useState } from 'react';

import { userApi } from '../entity/users';
import {
  AdminPasswordComponent,
  AuthNavbarComponent,
  SignInComponent,
  SignUpComponent,
} from '../features/users';

export function AuthPageComponent() {
  const [isAdminHasPassword, setIsAdminHasPassword] = useState(false);
  const [authMode, setAuthMode] = useState<'signIn' | 'signUp'>('signUp');
  const [isLoading, setLoading] = useState(true);

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
    <div>
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
                ) : (
                  <SignInComponent onSwitchToSignUp={() => setAuthMode('signUp')} />
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

import { LoadingOutlined } from '@ant-design/icons';
import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router';

import { getOAuthRedirectUri } from '../constants';
import { userApi } from '../entity/users';

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState<string>('');

  useEffect(() => {
    const handleOAuthCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');

      const expectedState = sessionStorage.getItem('oauth_state');
      const provider = sessionStorage.getItem('oauth_provider');
      sessionStorage.removeItem('oauth_state');
      sessionStorage.removeItem('oauth_provider');

      if (!code) {
        setError('Authorization code not found');
        return;
      }

      if (!state || !expectedState || state !== expectedState || !provider) {
        setError('Invalid OAuth state parameter');
        return;
      }

      const redirectUri = getOAuthRedirectUri();

      try {
        await userApi.handleOAuthCallback({ provider, code, redirectUri });
        navigate('/');
      } catch (e) {
        setError((e as Error).message || 'OAuth authentication failed');
      }
    };

    handleOAuthCallback();
  }, [searchParams, navigate]);

  return (
    <div className="flex h-screen w-screen flex-col items-center justify-center">
      {error ? (
        <div>
          <div className="mb-4 text-center text-xl font-semibold text-red-600">
            Authentication Failed
          </div>
          <div className="text-center text-sm text-gray-600">{error}</div>
          <div className="mt-6 text-center">
            <button
              type="button"
              onClick={() => navigate('/')}
              className="cursor-pointer font-medium text-blue-600 hover:text-blue-700"
            >
              Return to sign in
            </button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col items-center">
          <Spin indicator={<LoadingOutlined spin />} size="large" />
          <div className="mt-4 text-gray-600">Completing authentication...</div>
        </div>
      )}
    </div>
  );
}

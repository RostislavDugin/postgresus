import { LoginOutlined } from '@ant-design/icons';
import { Button, message } from 'antd';

import { GENERIC_OAUTH_PROVIDERS, getOAuthRedirectUri } from '../../../../constants';

export function GenericOAuthComponent() {
  if (GENERIC_OAUTH_PROVIDERS.length === 0) {
    return null;
  }

  const redirectUri = getOAuthRedirectUri();

  const handleLogin = (name: string, clientId: string, authUrl: string, scopes: string) => {
    try {
      const params = new URLSearchParams({
        client_id: clientId,
        redirect_uri: redirectUri,
        response_type: 'code',
        scope: scopes,
        state: name,
      });

      const url = `${authUrl}?${params.toString()}`;
      new URL(url);
      window.location.href = url;
    } catch (error) {
      message.error('Invalid OAuth configuration');
      console.error('Generic OAuth URL error:', error);
    }
  };

  return (
    <>
      {GENERIC_OAUTH_PROVIDERS.map((provider) => (
        <Button
          key={provider.name}
          icon={<LoginOutlined />}
          onClick={() =>
            handleLogin(provider.name, provider.clientId, provider.authUrl, provider.scopes)
          }
          className="w-full"
          size="large"
        >
          Continue with {provider.displayName}
        </Button>
      ))}
    </>
  );
}

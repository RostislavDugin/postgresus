import { LoginOutlined } from '@ant-design/icons';
import { Button, message } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { IS_OIDC_CONFIGURED, OIDC_DISPLAY_NAME, getOAuthRedirectUri } from '../../../../constants';
import { userApi } from '../../../../entity/users';
import { markOidcSignInStarted } from './oidcSignInMarker';

export function OidcSignInComponent() {
  const { t } = useTranslation();
  const [isStarting, setIsStarting] = useState(false);

  if (!IS_OIDC_CONFIGURED) {
    return null;
  }

  const handleOidcSignIn = async () => {
    setIsStarting(true);

    try {
      const { authorizationUrl } = await userApi.startOidcSignIn({
        redirectUri: getOAuthRedirectUri(),
      });

      markOidcSignInStarted();
      window.location.href = authorizationUrl;
    } catch {
      setIsStarting(false);
      message.error(t('users.oauth.oidcStartFailed'));
    }
  };

  return (
    <Button
      icon={<LoginOutlined />}
      onClick={handleOidcSignIn}
      loading={isStarting}
      className="w-full"
      size="large"
    >
      {OIDC_DISPLAY_NAME || t('users.oauth.continueWithSingleSignOn')}
    </Button>
  );
}

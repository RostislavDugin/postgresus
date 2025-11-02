import { GithubOutlined, GoogleOutlined } from '@ant-design/icons';
import { Button, message } from 'antd';

import {
  GITHUB_CLIENT_ID,
  GOOGLE_CLIENT_ID,
  getOAuthRedirectUri,
  isOAuthEnabled,
} from '../../../constants';

export function OauthComponent() {
  if (!isOAuthEnabled()) {
    return null;
  }

  const redirectUri = getOAuthRedirectUri();

  const handleGitHubLogin = () => {
    if (!GITHUB_CLIENT_ID) {
      message.error('GitHub OAuth is not configured');
      return;
    }

    try {
      const params = new URLSearchParams({
        client_id: GITHUB_CLIENT_ID,
        redirect_uri: redirectUri,
        state: 'github',
        scope: 'user:email',
      });

      const githubAuthUrl = `https://github.com/login/oauth/authorize?${params.toString()}`;

      // Validate URL is properly formed
      new URL(githubAuthUrl);
      window.location.href = githubAuthUrl;
    } catch (error) {
      message.error('Invalid OAuth configuration');
      console.error('GitHub OAuth URL error:', error);
    }
  };

  const handleGoogleLogin = () => {
    if (!GOOGLE_CLIENT_ID) {
      message.error('Google OAuth is not configured');
      return;
    }

    try {
      const params = new URLSearchParams({
        client_id: GOOGLE_CLIENT_ID,
        redirect_uri: redirectUri,
        response_type: 'code',
        scope: 'openid email profile',
        state: 'google',
      });

      const googleAuthUrl = `https://accounts.google.com/o/oauth2/v2/auth?${params.toString()}`;

      // Validate URL is properly formed
      new URL(googleAuthUrl);
      window.location.href = googleAuthUrl;
    } catch (error) {
      message.error('Invalid OAuth configuration');
      console.error('Google OAuth URL error:', error);
    }
  };

  return (
    <div className="mt-4">
      <div className="space-y-2">
        {GITHUB_CLIENT_ID && (
          <Button
            icon={<GithubOutlined />}
            onClick={handleGitHubLogin}
            className="w-full"
            size="large"
          >
            Continue with GitHub
          </Button>
        )}

        {GOOGLE_CLIENT_ID && (
          <Button
            icon={<GoogleOutlined />}
            onClick={handleGoogleLogin}
            className="w-full"
            size="large"
          >
            Continue with Google
          </Button>
        )}
      </div>
    </div>
  );
}

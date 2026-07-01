import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Button, Input } from 'antd';
import { type JSX, useState } from 'react';

import { useCloudflareTurnstile } from '../../../shared/hooks/useCloudflareTurnstile';

import {
  CLOUDFLARE_TURNSTILE_SITE_KEY,
  GENERIC_OAUTH_PROVIDERS,
  GITHUB_CLIENT_ID,
  GOOGLE_CLIENT_ID,
  IS_EMAIL_CONFIGURED,
} from '../../../constants';
import { userApi } from '../../../entity/users';
import { StringUtils } from '../../../shared/lib';
import { FormValidator } from '../../../shared/lib/FormValidator';
import { CloudflareTurnstileWidget } from '../../../shared/ui/CloudflareTurnstileWidget';
import { GenericOAuthComponent } from './oauth/GenericOAuthComponent';
import { GithubOAuthComponent } from './oauth/GithubOAuthComponent';
import { GoogleOAuthComponent } from './oauth/GoogleOAuthComponent';

interface SignInComponentProps {
  onSwitchToSignUp?: () => void;
  onSwitchToResetPassword?: () => void;
}

export function SignInComponent({
  onSwitchToSignUp,
  onSwitchToResetPassword,
}: SignInComponentProps): JSX.Element {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);

  const [signInError, setSignInError] = useState('');

  const { token, containerRef, resetCloudflareTurnstile } = useCloudflareTurnstile();

  const validateFieldsForSignIn = (): boolean => {
    if (!email) {
      setEmailError(true);
      return false;
    }

    if (!FormValidator.isValidEmail(email) && email !== 'admin') {
      setEmailError(true);
      return false;
    }

    if (!password) {
      setPasswordError(true);
      return false;
    }
    setPasswordError(false);

    return true;
  };

  const onSignIn = async () => {
    setSignInError('');

    if (validateFieldsForSignIn()) {
      setLoading(true);

      try {
        await userApi.signIn({
          email,
          password,
          cloudflareTurnstileToken: token,
        });
      } catch (e) {
        setSignInError(StringUtils.capitalizeFirstLetter((e as Error).message));
        resetCloudflareTurnstile();
      }

      setLoading(false);
    }
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">Sign in</div>

      <div className="mt-4">
        <div className="space-y-2">
          <GithubOAuthComponent />
          <GoogleOAuthComponent />
          <GenericOAuthComponent />
        </div>
      </div>

      {(GOOGLE_CLIENT_ID || GITHUB_CLIENT_ID || GENERIC_OAUTH_PROVIDERS.length > 0) && (
        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="bg-white px-2 text-gray-500 dark:bg-gray-900 dark:text-gray-400">
              or continue
            </span>
          </div>
        </div>
      )}

      <div className="my-1 text-xs font-semibold">Your email</div>
      <Input
        placeholder="your@email.com"
        value={email}
        onChange={(e) => {
          setEmailError(false);
          setEmail(e.currentTarget.value.trim().toLowerCase());
        }}
        status={isEmailError ? 'error' : undefined}
        type="email"
      />

      <div className="my-1 text-xs font-semibold">Password</div>
      <Input.Password
        placeholder="********"
        value={password}
        onChange={(e) => {
          setPasswordError(false);
          setPassword(e.currentTarget.value);
        }}
        status={passwordError ? 'error' : undefined}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{ visible: passwordVisible, onVisibleChange: setPasswordVisible }}
      />

      <div className="mt-3" />

      <CloudflareTurnstileWidget containerRef={containerRef} />

      <Button
        disabled={isLoading || (!!CLOUDFLARE_TURNSTILE_SITE_KEY && !token)}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSignIn();
        }}
        type="primary"
      >
        Sign in
      </Button>

      {signInError && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {signInError}
        </div>
      )}

      <div className="mt-4 text-center text-sm text-gray-600 dark:text-gray-400">
        Don&apos;t have an account?{' '}
        <button
          type="button"
          onClick={onSwitchToSignUp}
          className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
        >
          Sign up
        </button>
        <br />
        {IS_EMAIL_CONFIGURED && (
          <button
            type="button"
            onClick={onSwitchToResetPassword}
            className="cursor-pointer font-medium text-blue-600 hover:text-blue-700 dark:!text-blue-500"
          >
            Forgot password?
          </button>
        )}
      </div>

      <div className="mb-10" />
    </div>
  );
}

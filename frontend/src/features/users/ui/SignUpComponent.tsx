import { EyeInvisibleOutlined, EyeTwoTone } from '@ant-design/icons';
import { Button, Input } from 'antd';
import { type JSX, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { userApi } from '../../../entity/users';
import { FormValidator } from '../../../shared/lib/FormValidator';

export function SignUpComponent(): JSX.Element {
  const { t } = useTranslation('auth');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [confirmPassword, setConfirmPassword] = useState('');
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false);

  const [isLoading, setLoading] = useState(false);

  const [isEmailError, setEmailError] = useState(false);
  const [passwordError, setPasswordError] = useState(false);
  const [confirmPasswordError, setConfirmPasswordError] = useState(false);

  const [signUpError, setSignUpError] = useState('');

  const validateFieldsForSignUp = (): boolean => {
    if (!email) {
      setEmailError(true);
      return false;
    }

    if (!FormValidator.isValidEmail(email)) {
      setEmailError(true);
      return false;
    }

    if (!password) {
      setPasswordError(true);
      return false;
    }

    if (password.length < 8) {
      setPasswordError(true);
      alert(t('validation.password_min_length'));
      return false;
    }
    setPasswordError(false);

    if (!confirmPassword) {
      setConfirmPasswordError(true);
      return false;
    }
    if (password !== confirmPassword) {
      setConfirmPasswordError(true);
      return false;
    }
    setConfirmPasswordError(false);

    return true;
  };

  const onSignUp = async () => {
    setSignUpError('');

    if (validateFieldsForSignUp()) {
      setLoading(true);

      try {
        await userApi.signUp({
          email,
          password,
        });
        await userApi.signIn({ email, password });
      } catch (e) {
        setSignUpError((e as Error).message);
      }
    }

    setLoading(false);
  };

  return (
    <div className="w-full max-w-[300px]">
      <div className="mb-5 text-center text-2xl font-bold">{t('sign_up.title')}</div>

      <div className="my-1 text-xs font-semibold">{t('sign_up.email_label')}</div>
      <Input
        placeholder={t('sign_up.email_placeholder')}
        value={email}
        onChange={(e) => {
          setEmailError(false);
          setEmail(e.currentTarget.value.trim().toLowerCase());
        }}
        status={isEmailError ? 'error' : undefined}
        type="email"
      />

      <div className="my-1 text-xs font-semibold">{t('sign_up.password_label')}</div>
      <Input.Password
        placeholder={t('sign_up.password_placeholder')}
        value={password}
        onChange={(e) => {
          setPasswordError(false);
          setPassword(e.currentTarget.value);
        }}
        status={passwordError ? 'error' : undefined}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{ visible: passwordVisible, onVisibleChange: setPasswordVisible }}
      />

      <div className="my-1 text-xs font-semibold">{t('sign_up.confirm_password_label')}</div>
      <Input.Password
        placeholder={t('sign_up.confirm_password_placeholder')}
        value={confirmPassword}
        status={confirmPasswordError ? 'error' : undefined}
        onChange={(e) => {
          setConfirmPasswordError(false);
          setConfirmPassword(e.currentTarget.value);
        }}
        iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
        visibilityToggle={{
          visible: confirmPasswordVisible,
          onVisibleChange: setConfirmPasswordVisible,
        }}
      />

      <div className="mt-3" />

      <Button
        disabled={isLoading}
        loading={isLoading}
        className="w-full"
        onClick={() => {
          onSignUp();
        }}
        type="primary"
      >
        {t('sign_up.button')}
      </Button>

      {signUpError && (
        <div className="mt-3 flex justify-center text-center text-sm text-red-600">
          {signUpError}
        </div>
      )}
    </div>
  );
}

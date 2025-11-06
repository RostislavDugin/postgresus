import { useEffect, useState } from 'react';
import { BrowserRouter, Route } from 'react-router';
import { Routes } from 'react-router';
import { ConfigProvider } from 'antd';
import { useTranslation } from 'react-i18next';
import enUS from 'antd/locale/en_US';
import zhCN from 'antd/locale/zh_CN';

import { userApi } from './entity/users';
import { OauthStorageComponent } from './features/storages/OauthStorageComponent';
import { AuthPageComponent } from './pages/AuthPageComponent';
import { MainScreenComponent } from './widgets/main/MainScreenComponent';

const antdLocales: Record<string, any> = {
  en: enUS,
  zh: zhCN,
};

function App() {
  const { i18n } = useTranslation();
  const [isAuthorized, setIsAuthorized] = useState(false);

  const currentLocale = antdLocales[i18n.language] || enUS;

  useEffect(() => {
    const isAuthorized = userApi.isAuthorized();
    setIsAuthorized(isAuthorized);

    userApi.addAuthListener(() => {
      setIsAuthorized(userApi.isAuthorized());
    });
  }, []);

  return (
    <ConfigProvider locale={currentLocale}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={!isAuthorized ? <AuthPageComponent /> : <MainScreenComponent />} />
          <Route
            path="/storages/google-oauth"
            element={!isAuthorized ? <AuthPageComponent /> : <OauthStorageComponent />}
          />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;

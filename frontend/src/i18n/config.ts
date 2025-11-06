import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import dayjs from 'dayjs';

// 导入翻译文件
import commonEN from './locales/en/common.json';
import authEN from './locales/en/auth.json';
import databaseEN from './locales/en/database.json';
import storageEN from './locales/en/storage.json';
import notifierEN from './locales/en/notifier.json';
import backupEN from './locales/en/backup.json';
import restoreEN from './locales/en/restore.json';
import healthcheckEN from './locales/en/healthcheck.json';

import commonZH from './locales/zh/common.json';
import authZH from './locales/zh/auth.json';
import databaseZH from './locales/zh/database.json';
import storageZH from './locales/zh/storage.json';
import notifierZH from './locales/zh/notifier.json';
import backupZH from './locales/zh/backup.json';
import restoreZH from './locales/zh/restore.json';
import healthcheckZH from './locales/zh/healthcheck.json';

const resources = {
  en: {
    common: commonEN,
    auth: authEN,
    database: databaseEN,
    storage: storageEN,
    notifier: notifierEN,
    backup: backupEN,
    restore: restoreEN,
    healthcheck: healthcheckEN,
  },
  zh: {
    common: commonZH,
    auth: authZH,
    database: databaseZH,
    storage: storageZH,
    notifier: notifierZH,
    backup: backupZH,
    restore: restoreZH,
    healthcheck: healthcheckZH,
  },
};

i18n
  .use(LanguageDetector) // 自动检测用户语言
  .use(initReactI18next) // 集成 React
  .init({
    resources,
    fallbackLng: 'en', // 默认语言
    defaultNS: 'common', // 默认命名空间
    ns: ['common', 'auth', 'database', 'storage', 'notifier', 'backup', 'restore', 'healthcheck'],

    interpolation: {
      escapeValue: false, // React 已经转义
    },

    detection: {
      // 语言检测优先级
      order: ['localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      lookupLocalStorage: 'i18nextLng',
    },

    react: {
      useSuspense: false, // 避免 Suspense 组件
    },
  });

// 同步 dayjs locale
const syncDayjsLocale = (lng: string) => {
  if (lng === 'zh' || lng.startsWith('zh-')) {
    dayjs.locale('zh-cn');
  } else {
    dayjs.locale('en');
  }
};

// 初始设置 dayjs locale
syncDayjsLocale(i18n.language);

// 监听语言变化
i18n.on('languageChanged', (lng) => {
  syncDayjsLocale(lng);
});

export default i18n;

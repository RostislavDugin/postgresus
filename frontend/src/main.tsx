import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import utc from 'dayjs/plugin/utc';
import 'dayjs/locale/zh-cn'; // 导入中文locale
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import './index.css';
import './i18n/config'; // 导入 i18n 配置

import App from './App.tsx';

dayjs.extend(utc);
dayjs.extend(relativeTime);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

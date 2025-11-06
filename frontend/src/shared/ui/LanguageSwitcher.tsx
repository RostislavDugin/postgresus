import React from 'react';
import { useTranslation } from 'react-i18next';
import { Select } from 'antd';
import { GlobalOutlined } from '@ant-design/icons';

const languages = [
  { code: 'en', label: 'English', flag: '🇺🇸' },
  { code: 'zh', label: '中文', flag: '🇨🇳' },
];

export const LanguageSwitcher: React.FC = () => {
  const { i18n } = useTranslation();

  const handleChange = (lang: string) => {
    i18n.changeLanguage(lang);
  };

  return (
    <Select
      value={i18n.language}
      onChange={handleChange}
      style={{ width: 150 }}
      suffixIcon={<GlobalOutlined />}
    >
      {languages.map((lang) => (
        <Select.Option key={lang.code} value={lang.code}>
          <span>
            {lang.flag} {lang.label}
          </span>
        </Select.Option>
      ))}
    </Select>
  );
};

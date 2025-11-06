import { InfoCircleOutlined } from '@ant-design/icons';
import { Button, Input, Select, Switch, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import type { Notifier } from '../../../../../entity/notifiers';
import { WebhookMethod } from '../../../../../entity/notifiers/models/webhook/WebhookMethod';

const { TextArea } = Input;

interface Props {
  notifier: Notifier;
  setNotifier: (notifier: Notifier) => void;
  setIsUnsaved: (isUnsaved: boolean) => void;
}

export function EditWebhookNotifierComponent({ notifier, setNotifier, setIsUnsaved }: Props) {
  const { t } = useTranslation(['notifier', 'common']);

  // 插入变量到模板
  const insertVariable = (variable: string) => {
    const currentTemplate = notifier?.webhookNotifier?.customTemplate || '';
    setNotifier({
      ...notifier,
      webhookNotifier: {
        ...(notifier.webhookNotifier || {
          webhookUrl: '',
          webhookMethod: WebhookMethod.POST,
        }),
        customTemplate: currentTemplate + variable,
      },
    });
    setIsUnsaved(true);
  };

  // 应用企业微信模板
  const applyWeworkTemplate = () => {
    const weworkTemplate = `{
  "msgtype": "text",
  "text": {
    "content": "{{status}} 数据库【{{database_name}}】备份{{status_text}}\\n\\n⏱ 耗时：{{duration}}\\n💾 大小：{{size}}"
  }
}`;
    setNotifier({
      ...notifier,
      webhookNotifier: {
        ...(notifier.webhookNotifier || {
          webhookUrl: '',
          webhookMethod: WebhookMethod.POST,
        }),
        customTemplate: weworkTemplate,
      },
    });
    setIsUnsaved(true);
  };

  // 生成示例请求
  const getExampleRequest = () => {
    // 使用细粒度变量作为示例
    const exampleVars = {
      status: '✅',
      status_text: 'success',
      database_name: 'MyDatabase',
      duration: '2m 17s',
      size: '1.7GB',
      error: '',
      // 向后兼容的旧变量
      heading: '✅ Backup completed for database "MyDatabase"',
      message: 'Backup completed successfully in 2m 17s.\\nCompressed backup size: 1.7GB',
    };

    if (notifier?.webhookNotifier?.customTemplate) {
      // 使用自定义模板
      let customBody = notifier.webhookNotifier.customTemplate;

      // 替换所有变量
      Object.entries(exampleVars).forEach(([key, value]) => {
        customBody = customBody.replace(new RegExp(`\\{\\{${key}\\}\\}`, 'g'), value);
      });

      return notifier?.webhookNotifier?.webhookMethod === WebhookMethod.GET
        ? `GET ${notifier?.webhookNotifier?.webhookUrl}?${customBody}`
        : `POST ${notifier?.webhookNotifier?.webhookUrl}
Content-Type: application/json

${customBody}`;
    } else {
      // 使用默认格式
      return notifier?.webhookNotifier?.webhookMethod === WebhookMethod.GET
        ? `GET ${notifier?.webhookNotifier?.webhookUrl}?heading=${exampleVars.heading}&message=${exampleVars.message}`
        : `POST ${notifier?.webhookNotifier?.webhookUrl}
Content-Type: application/json

{
  "heading": "${exampleVars.heading}",
  "message": "${exampleVars.message}"
}`;
    }
  };

  return (
    <>
      <div className="flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.webhook_url_label')}</div>

        <div className="w-[250px]">
          <Input
            value={notifier?.webhookNotifier?.webhookUrl || ''}
            onChange={(e) => {
              setNotifier({
                ...notifier,
                webhookNotifier: {
                  ...(notifier.webhookNotifier || { webhookMethod: WebhookMethod.POST }),
                  webhookUrl: e.target.value.trim(),
                },
              });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            placeholder={t('notifier:form.webhook_url_placeholder')}
          />
        </div>
      </div>

      <div className="mt-1 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.webhook_method_label')}</div>

        <div className="w-[250px]">
          <Select
            value={notifier?.webhookNotifier?.webhookMethod || WebhookMethod.POST}
            onChange={(value) => {
              setNotifier({
                ...notifier,
                webhookNotifier: {
                  ...(notifier.webhookNotifier || { webhookUrl: '' }),
                  webhookMethod: value,
                },
              });
              setIsUnsaved(true);
            }}
            size="small"
            className="w-full"
            options={[
              { value: WebhookMethod.POST, label: 'POST' },
              { value: WebhookMethod.GET, label: 'GET' },
            ]}
          />
        </div>

        <Tooltip
          className="cursor-pointer"
          title={t('notifier:form.webhook_method_tooltip')}
        >
          <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      {/* 自定义模板开关 */}
      <div className="mt-3 flex items-center">
        <div className="w-[130px] min-w-[130px]">{t('notifier:form.use_custom_template_label')}</div>
        <Switch
          checked={!!notifier?.webhookNotifier?.customTemplate}
          onChange={(checked) => {
            setNotifier({
              ...notifier,
              webhookNotifier: {
                ...(notifier.webhookNotifier || {
                  webhookUrl: '',
                  webhookMethod: WebhookMethod.POST,
                }),
                customTemplate: checked
                  ? notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST
                    ? '{\n  "heading": "{{heading}}",\n  "message": "{{message}}"\n}'
                    : 'heading={{heading}}&message={{message}}'
                  : undefined,
              },
            });
            setIsUnsaved(true);
          }}
          size="small"
        />
        <Tooltip title={t('notifier:form.use_custom_template_tooltip')}>
          <InfoCircleOutlined className="ml-2 cursor-pointer" style={{ color: 'gray' }} />
        </Tooltip>
      </div>

      {/* 自定义模板编辑器 */}
      {notifier?.webhookNotifier?.customTemplate !== undefined && (
        <div className="mt-2">
          <div className="mb-2">
            <div className="mb-1 text-sm text-gray-600">
              {t('notifier:form.custom_template_label')}
            </div>
            <div className="flex flex-wrap gap-2">
              <Tooltip title={t('notifier:form.status_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{status}}')}
                >
                  {t('notifier:form.insert_status_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.status_text_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{status_text}}')}
                >
                  {t('notifier:form.insert_status_text_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.database_name_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{database_name}}')}
                >
                  {t('notifier:form.insert_database_name_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.duration_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{duration}}')}
                >
                  {t('notifier:form.insert_duration_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.size_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{size}}')}
                >
                  {t('notifier:form.insert_size_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.error_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{error}}')}
                >
                  {t('notifier:form.insert_error_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.heading_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{heading}}')}
                >
                  {t('notifier:form.insert_heading_variable')}
                </Button>
              </Tooltip>
              <Tooltip title={t('notifier:form.message_variable_tooltip')}>
                <Button
                  size="small"
                  type="link"
                  onClick={() => insertVariable('{{message}}')}
                >
                  {t('notifier:form.insert_message_variable')}
                </Button>
              </Tooltip>
            </div>
          </div>

          {/* 模板预设按钮 */}
          {notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST && (
            <div className="mb-2 flex items-center gap-2">
              <span className="text-xs text-gray-500">{t('notifier:form.template_presets')}:</span>
              <Button
                size="small"
                onClick={applyWeworkTemplate}
              >
                {t('notifier:form.use_wework_template')}
              </Button>
            </div>
          )}

          <TextArea
            value={notifier?.webhookNotifier?.customTemplate || ''}
            onChange={(e) => {
              setNotifier({
                ...notifier,
                webhookNotifier: {
                  ...(notifier.webhookNotifier || {
                    webhookUrl: '',
                    webhookMethod: WebhookMethod.POST,
                  }),
                  customTemplate: e.target.value,
                },
              });
              setIsUnsaved(true);
            }}
            placeholder={
              notifier?.webhookNotifier?.webhookMethod === WebhookMethod.POST
                ? t('notifier:form.custom_template_placeholder_post')
                : t('notifier:form.custom_template_placeholder_get')
            }
            rows={6}
            className="font-mono text-sm"
          />

          <div className="mt-1 text-xs text-gray-500">
            {t('notifier:form.custom_template_help')}
          </div>
        </div>
      )}

      {/* 示例请求 */}
      {notifier?.webhookNotifier?.webhookUrl && (
        <div className="mt-3">
          <div className="mb-1 font-medium text-sm">{t('notifier:form.example_request')}</div>

          <div className="rounded bg-gray-100 p-2 px-3 font-mono text-sm break-all whitespace-pre-line">
            {getExampleRequest()}
          </div>
        </div>
      )}
    </>
  );
}

package webhook_notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type WebhookNotifier struct {
	NotifierID     uuid.UUID     `json:"notifierId"     gorm:"primaryKey;column:notifier_id"`
	WebhookURL     string        `json:"webhookUrl"     gorm:"not null;column:webhook_url"`
	WebhookMethod  WebhookMethod `json:"webhookMethod"  gorm:"not null;column:webhook_method"`
	CustomTemplate *string       `json:"customTemplate" gorm:"column:custom_template;type:text"`
}

func (t *WebhookNotifier) TableName() string {
	return "webhook_notifiers"
}

// replaceVariablesInJSON 递归处理JSON数据，替换所有字符串中的变量并转换转义序列
func replaceVariablesInJSON(data interface{}, vars map[string]string) interface{} {
	switch v := data.(type) {
	case string:
		// 替换变量
		result := v
		for key, value := range vars {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}
		// 转换转义序列（字面的\n转为实际换行符）
		result = strings.ReplaceAll(result, "\\n", "\n")
		result = strings.ReplaceAll(result, "\\t", "\t")
		result = strings.ReplaceAll(result, "\\r", "\r")
		return result
	case map[string]interface{}:
		// 递归处理 map
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = replaceVariablesInJSON(val, vars)
		}
		return result
	case []interface{}:
		// 递归处理 array
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = replaceVariablesInJSON(val, vars)
		}
		return result
	default:
		// 其他类型（数字、布尔值等）直接返回
		return v
	}
}

func (t *WebhookNotifier) Validate() error {
	if t.WebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	if t.WebhookMethod == "" {
		return errors.New("webhook method is required")
	}

	return nil
}

func (t *WebhookNotifier) Send(logger *slog.Logger, vars map[string]string) error {
	// Get heading and message for backward compatibility
	heading := vars["heading"]
	message := vars["message"]

	// 添加调试日志 - 输入参数
	logger.Info("Webhook Send called",
		"heading", heading,
		"message", message,
		"headingLength", len(heading),
		"messageLength", len(message),
		"hasCustomTemplate", t.CustomTemplate != nil && *t.CustomTemplate != "",
		"varsCount", len(vars),
	)

	// 替换模板中的变量
	replaceVariables := func(template string) string {
		result := template
		for key, value := range vars {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}

		// 将字面字符串 \n \t \r 转换为实际的转义字符
		// 这样JSON序列化后会正确显示为 \n（转义序列）而不是 \\n（字面字符）
		result = strings.ReplaceAll(result, "\\n", "\n")
		result = strings.ReplaceAll(result, "\\t", "\t")
		result = strings.ReplaceAll(result, "\\r", "\r")

		logger.Info("Template replacement",
			"originalTemplate", template,
			"afterReplacement", result,
		)

		return result
	}

	switch t.WebhookMethod {
	case WebhookMethodGET:
		var reqURL string
		if t.CustomTemplate != nil && *t.CustomTemplate != "" {
			// 使用自定义模板
			customQuery := replaceVariables(*t.CustomTemplate)
			reqURL = fmt.Sprintf("%s?%s", t.WebhookURL, customQuery)

			logger.Info("Webhook GET with custom template",
				"url", reqURL,
			)
		} else {
			// 使用默认格式
			reqURL = fmt.Sprintf("%s?heading=%s&message=%s",
				t.WebhookURL,
				url.QueryEscape(heading),
				url.QueryEscape(message),
			)

			logger.Info("Webhook GET with default format",
				"url", reqURL,
			)
		}

		resp, err := http.Get(reqURL)
		if err != nil {
			return fmt.Errorf("failed to send GET webhook: %w", err)
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				logger.Error("failed to close response body", "error", cerr)
			}
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf(
				"webhook GET returned status: %s, body: %s",
				resp.Status,
				string(body),
			)
		}

		return nil

	case WebhookMethodPOST:
		var body []byte
		var err error

		if t.CustomTemplate != nil && *t.CustomTemplate != "" {
			// 使用自定义模板
			logger.Info("Using custom template",
				"template", *t.CustomTemplate,
			)

			// 解析JSON模板
			var templateData interface{}
			if err := json.Unmarshal([]byte(*t.CustomTemplate), &templateData); err != nil {
				return fmt.Errorf("failed to parse custom template as JSON: %w", err)
			}

			// 递归替换所有字符串中的变量
			processedData := replaceVariablesInJSON(templateData, vars)

			// 重新序列化为JSON
			body, err = json.Marshal(processedData)
			if err != nil {
				return fmt.Errorf("failed to marshal processed template: %w", err)
			}

			logger.Info("Webhook POST body prepared",
				"bodyLength", len(body),
				"bodyContent", string(body),
			)
		} else {
			// 使用默认格式
			payload := map[string]string{
				"heading": heading,
				"message": message,
			}
			body, err = json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal webhook payload: %w", err)
			}

			logger.Info("Webhook POST with default format",
				"bodyLength", len(body),
				"bodyContent", string(body),
			)
		}

		logger.Info("Sending POST request",
			"url", t.WebhookURL,
			"contentType", "application/json",
		)

		resp, err := http.Post(t.WebhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			logger.Error("Failed to send POST webhook",
				"error", err,
				"url", t.WebhookURL,
			)
			return fmt.Errorf("failed to send POST webhook: %w", err)
		}

		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				logger.Error("failed to close response body", "error", cerr)
			}
		}()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			logger.Error("Webhook returned error status",
				"statusCode", resp.StatusCode,
				"status", resp.Status,
				"responseBody", string(body),
			)
			return fmt.Errorf(
				"webhook POST returned status: %s, body: %s",
				resp.Status,
				string(body),
			)
		}

		logger.Info("Webhook sent successfully",
			"statusCode", resp.StatusCode,
		)

		return nil

	default:
		return fmt.Errorf("unsupported webhook method: %s", t.WebhookMethod)
	}
}

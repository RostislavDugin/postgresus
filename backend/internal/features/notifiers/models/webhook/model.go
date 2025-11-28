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
	"postgresus-backend/internal/util/encryption"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WebhookHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WebhookNotifier struct {
	NotifierID    uuid.UUID     `json:"notifierId"    gorm:"primaryKey;column:notifier_id"`
	WebhookURL    string        `json:"webhookUrl"    gorm:"not null;column:webhook_url"`
	WebhookMethod WebhookMethod `json:"webhookMethod" gorm:"not null;column:webhook_method"`
	BodyTemplate  *string       `json:"bodyTemplate"  gorm:"column:body_template;type:text"`
	HeadersJSON   string        `json:"-"             gorm:"column:headers;type:text"`

	Headers []WebhookHeader `json:"headers" gorm:"-"`
}

func (t *WebhookNotifier) TableName() string {
	return "webhook_notifiers"
}

func (t *WebhookNotifier) BeforeSave(_ *gorm.DB) error {
	if len(t.Headers) > 0 {
		data, err := json.Marshal(t.Headers)

		if err != nil {
			return err
		}

		t.HeadersJSON = string(data)
	} else {
		t.HeadersJSON = "[]"
	}

	return nil
}

func (t *WebhookNotifier) AfterFind(_ *gorm.DB) error {
	if t.HeadersJSON != "" {
		if err := json.Unmarshal([]byte(t.HeadersJSON), &t.Headers); err != nil {
			return err
		}
	}

	return nil
}

func (t *WebhookNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if t.WebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	if t.WebhookMethod == "" {
		return errors.New("webhook method is required")
	}

	return nil
}

func (t *WebhookNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	heading string,
	message string,
) error {
	webhookURL, err := encryptor.Decrypt(t.NotifierID, t.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to decrypt webhook URL: %w", err)
	}

	switch t.WebhookMethod {
	case WebhookMethodGET:
		return t.sendGET(webhookURL, heading, message, logger)
	case WebhookMethodPOST:
		return t.sendPOST(webhookURL, heading, message, logger)
	default:
		return fmt.Errorf("unsupported webhook method: %s", t.WebhookMethod)
	}
}

func (t *WebhookNotifier) HideSensitiveData() {
}

func (t *WebhookNotifier) Update(incoming *WebhookNotifier) {
	t.WebhookURL = incoming.WebhookURL
	t.WebhookMethod = incoming.WebhookMethod
	t.BodyTemplate = incoming.BodyTemplate
	t.Headers = incoming.Headers
}

func (t *WebhookNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	if t.WebhookURL != "" {
		encrypted, err := encryptor.Encrypt(t.NotifierID, t.WebhookURL)

		if err != nil {
			return fmt.Errorf("failed to encrypt webhook URL: %w", err)
		}

		t.WebhookURL = encrypted
	}

	return nil
}

func (t *WebhookNotifier) sendGET(webhookURL, heading, message string, logger *slog.Logger) error {
	reqURL := fmt.Sprintf("%s?heading=%s&message=%s",
		webhookURL,
		url.QueryEscape(heading),
		url.QueryEscape(message),
	)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request: %w", err)
	}

	t.applyHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
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
}

func (t *WebhookNotifier) sendPOST(webhookURL, heading, message string, logger *slog.Logger) error {
	body := t.buildRequestBody(heading, message)

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}

	hasContentType := false

	for _, h := range t.Headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
			break
		}
	}

	if !hasContentType {
		req.Header.Set("Content-Type", "application/json")
	}

	t.applyHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST webhook: %w", err)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logger.Error("failed to close response body", "error", cerr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"webhook POST returned status: %s, body: %s",
			resp.Status,
			string(respBody),
		)
	}

	return nil
}

func (t *WebhookNotifier) buildRequestBody(heading, message string) []byte {
	if t.BodyTemplate != nil && *t.BodyTemplate != "" {
		result := *t.BodyTemplate
		result = strings.ReplaceAll(result, "{{heading}}", heading)
		result = strings.ReplaceAll(result, "{{message}}", message)
		return []byte(result)
	}

	payload := map[string]string{
		"heading": heading,
		"message": message,
	}
	body, _ := json.Marshal(payload)

	return body
}

func (t *WebhookNotifier) applyHeaders(req *http.Request) {
	for _, h := range t.Headers {
		if h.Key != "" {
			req.Header.Set(h.Key, h.Value)
		}
	}
}

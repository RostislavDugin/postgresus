package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=EMAIL;TELEGRAM;WEBHOOK;SLACK;DISCORD;TEAMS
type NotifierType string

const (
	NotifierTypeEmail    NotifierType = "EMAIL"
	NotifierTypeTelegram NotifierType = "TELEGRAM"
	NotifierTypeWebhook  NotifierType = "WEBHOOK"
	NotifierTypeSlack    NotifierType = "SLACK"
	NotifierTypeDiscord  NotifierType = "DISCORD"
	NotifierTypeTeams    NotifierType = "TEAMS"
)

type NotifierSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Type NotifierType `json:"type"`

	Discord  *DiscordNotifierSpec  `json:"discord,omitempty"`
	Slack    *SlackNotifierSpec    `json:"slack,omitempty"`
	Telegram *TelegramNotifierSpec `json:"telegram,omitempty"`
	Email    *EmailNotifierSpec    `json:"email,omitempty"`
	Webhook  *WebhookNotifierSpec  `json:"webhook,omitempty"`
	Teams    *TeamsNotifierSpec    `json:"teams,omitempty"`
}

type DiscordNotifierSpec struct {
	WebhookURLSecretRef SecretKeyRef `json:"webhookURLSecretRef"`
}

type SlackNotifierSpec struct {
	BotTokenSecretRef SecretKeyRef `json:"botTokenSecretRef"`
	TargetChatID      string       `json:"targetChatId"`
}

type TelegramNotifierSpec struct {
	BotTokenSecretRef SecretKeyRef `json:"botTokenSecretRef"`
	TargetChatID      string       `json:"targetChatId"`
	ThreadID          *int64       `json:"threadId,omitempty"`
}

type EmailNotifierSpec struct {
	TargetEmail           string        `json:"targetEmail"`
	SMTPHost              string        `json:"smtpHost"`
	SMTPPort              int           `json:"smtpPort"`
	SMTPUser              string        `json:"smtpUser,omitempty"`
	SMTPPasswordSecretRef *SecretKeyRef `json:"smtpPasswordSecretRef,omitempty"`
	From                  string        `json:"from,omitempty"`
	// +kubebuilder:default=false
	IsInsecureSkipVerify bool `json:"isInsecureSkipVerify,omitempty"`
}

type WebhookNotifierSpec struct {
	WebhookURL string `json:"webhookUrl"`
	// +kubebuilder:validation:Enum=POST;GET;PUT
	WebhookMethod string          `json:"webhookMethod"`
	BodyTemplate  string          `json:"bodyTemplate,omitempty"`
	Headers       []WebhookHeader `json:"headers,omitempty"`
}

type WebhookHeader struct {
	Key            string        `json:"key"`
	ValueSecretRef *SecretKeyRef `json:"valueSecretRef,omitempty"`
	Value          string        `json:"value,omitempty"`
}

type TeamsNotifierSpec struct {
	WebhookURLSecretRef SecretKeyRef `json:"webhookURLSecretRef"`
}

type NotifierStatus struct {
	// ID assigned by the databasus API.
	ID string `json:"id,omitempty"`
	// ObservedGeneration is the most recent generation observed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Notifier is the Schema for the notifiers API.
type Notifier struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotifierSpec   `json:"spec"`
	Status NotifierStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NotifierList contains a list of Notifier.
type NotifierList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notifier `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notifier{}, &NotifierList{})
}

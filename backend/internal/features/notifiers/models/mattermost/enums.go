package mattermost_notifier

type DeliveryMode string

const (
	DeliveryModeWebhook DeliveryMode = "WEBHOOK"
	DeliveryModeBot     DeliveryMode = "BOT"
)

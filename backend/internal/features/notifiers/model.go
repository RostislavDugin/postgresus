package notifiers

import (
	"errors"
	"log/slog"
	discord_notifier "postgresus-backend/internal/features/notifiers/models/discord"
	"postgresus-backend/internal/features/notifiers/models/email_notifier"
	slack_notifier "postgresus-backend/internal/features/notifiers/models/slack"
	teams_notifier "postgresus-backend/internal/features/notifiers/models/teams"
	telegram_notifier "postgresus-backend/internal/features/notifiers/models/telegram"
	webhook_notifier "postgresus-backend/internal/features/notifiers/models/webhook"

	"github.com/google/uuid"
)

type Notifier struct {
	ID            uuid.UUID    `json:"id"            gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID        uuid.UUID    `json:"userId"        gorm:"column:user_id;not null;type:uuid;index"`
	Name          string       `json:"name"          gorm:"column:name;not null;type:varchar(255)"`
	NotifierType  NotifierType `json:"notifierType"  gorm:"column:notifier_type;not null;type:varchar(50)"`
	LastSendError *string      `json:"lastSendError" gorm:"column:last_send_error;type:text"`

	// specific notifier
	TelegramNotifier *telegram_notifier.TelegramNotifier `json:"telegramNotifier"        gorm:"foreignKey:NotifierID"`
	EmailNotifier    *email_notifier.EmailNotifier       `json:"emailNotifier"           gorm:"foreignKey:NotifierID"`
	WebhookNotifier  *webhook_notifier.WebhookNotifier   `json:"webhookNotifier"         gorm:"foreignKey:NotifierID"`
	SlackNotifier    *slack_notifier.SlackNotifier       `json:"slackNotifier"           gorm:"foreignKey:NotifierID"`
	DiscordNotifier  *discord_notifier.DiscordNotifier   `json:"discordNotifier"         gorm:"foreignKey:NotifierID"`
	TeamsNotifier    *teams_notifier.TeamsNotifier       `json:"teamsNotifier,omitempty" gorm:"foreignKey:NotifierID;constraint:OnDelete:CASCADE"`
}

func (n *Notifier) TableName() string {
	return "notifiers"
}

func (n *Notifier) Validate() error {
	if n.Name == "" {
		return errors.New("name is required")
	}

	return n.getSpecificNotifier().Validate()
}

func (n *Notifier) Send(logger *slog.Logger, vars map[string]string) error {
	err := n.getSpecificNotifier().Send(logger, vars)

	if err != nil {
		lastSendError := err.Error()
		n.LastSendError = &lastSendError
	} else {
		n.LastSendError = nil
	}

	return err
}

func (n *Notifier) getSpecificNotifier() NotificationSender {
	switch n.NotifierType {
	case NotifierTypeTelegram:
		return n.TelegramNotifier
	case NotifierTypeEmail:
		return n.EmailNotifier
	case NotifierTypeWebhook:
		return n.WebhookNotifier
	case NotifierTypeSlack:
		return n.SlackNotifier
	case NotifierTypeDiscord:
		return n.DiscordNotifier
	case NotifierTypeTeams:
		return n.TeamsNotifier
	default:
		panic("unknown notifier type: " + string(n.NotifierType))
	}
}

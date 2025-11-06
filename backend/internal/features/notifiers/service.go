package notifiers

import (
	"errors"
	"log/slog"
	users_models "postgresus-backend/internal/features/users/models"

	"github.com/google/uuid"
)

type NotifierService struct {
	notifierRepository *NotifierRepository
	logger             *slog.Logger
}

func (s *NotifierService) SaveNotifier(
	user *users_models.User,
	notifier *Notifier,
) error {
	if notifier.ID != uuid.Nil {
		existingNotifier, err := s.notifierRepository.FindByID(notifier.ID)
		if err != nil {
			return err
		}

		if existingNotifier.UserID != user.ID {
			return errors.New("you have not access to this notifier")
		}

		notifier.UserID = existingNotifier.UserID
	} else {
		notifier.UserID = user.ID
	}

	_, err := s.notifierRepository.Save(notifier)
	if err != nil {
		return err
	}

	return nil
}

func (s *NotifierService) DeleteNotifier(
	user *users_models.User,
	notifierID uuid.UUID,
) error {
	notifier, err := s.notifierRepository.FindByID(notifierID)
	if err != nil {
		return err
	}

	if notifier.UserID != user.ID {
		return errors.New("you have not access to this notifier")
	}

	return s.notifierRepository.Delete(notifier)
}

func (s *NotifierService) GetNotifier(
	user *users_models.User,
	id uuid.UUID,
) (*Notifier, error) {
	notifier, err := s.notifierRepository.FindByID(id)
	if err != nil {
		return nil, err
	}

	if notifier.UserID != user.ID {
		return nil, errors.New("you have not access to this notifier")
	}

	return notifier, nil
}

func (s *NotifierService) GetNotifiers(
	user *users_models.User,
) ([]*Notifier, error) {
	return s.notifierRepository.FindByUserID(user.ID)
}

func (s *NotifierService) SendTestNotification(
	user *users_models.User,
	notifierID uuid.UUID,
) error {
	notifier, err := s.notifierRepository.FindByID(notifierID)
	if err != nil {
		return err
	}

	if notifier.UserID != user.ID {
		return errors.New("you have not access to this notifier")
	}

	// 提供所有细粒度变量的示例值
	vars := map[string]string{
		"status":        "✅",
		"status_text":   "success",
		"database_name": "TestDatabase",
		"duration":      "2m 17s",
		"size":          "1.7GB",
		"error":         "",
		// 向后兼容的旧变量
		"heading": "✅ Test message",
		"message": "This is a test notification with sample data.\nDuration: 2m 17s\nSize: 1.7GB",
	}
	err = notifier.Send(s.logger, vars)
	if err != nil {
		return err
	}

	_, err = s.notifierRepository.Save(notifier)
	if err != nil {
		return err
	}

	return nil
}

func (s *NotifierService) SendTestNotificationToNotifier(
	notifier *Notifier,
) error {
	vars := map[string]string{
		"heading": "Test message",
		"message": "This is a test message",
	}
	return notifier.Send(s.logger, vars)
}

func (s *NotifierService) SendNotification(
	notifier *Notifier,
	vars map[string]string,
) {
	// Truncate message to 2000 characters if it's too long
	if message, ok := vars["message"]; ok {
		messageRunes := []rune(message)
		if len(messageRunes) > 2000 {
			vars["message"] = string(messageRunes[:2000])
		}
	}

	notifiedFromDb, err := s.notifierRepository.FindByID(notifier.ID)
	if err != nil {
		return
	}

	err = notifiedFromDb.Send(s.logger, vars)
	if err != nil {
		errMsg := err.Error()
		notifiedFromDb.LastSendError = &errMsg

		_, err = s.notifierRepository.Save(notifiedFromDb)
		if err != nil {
			s.logger.Error("Failed to save notifier", "error", err)
		}
	}

	notifiedFromDb.LastSendError = nil
	_, err = s.notifierRepository.Save(notifiedFromDb)
	if err != nil {
		s.logger.Error("Failed to save notifier", "error", err)
	}
}

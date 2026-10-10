package telegram_notifier

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const officialBotAPIURL = "https://api.telegram.org"

// Send validates again because a direct test send of an unsaved notifier never reaches Validate.
// Concatenation rather than url.JoinPath keeps "./" and "../" in the prefix exactly as validated.
func (t *TelegramNotifier) getSendMessageURL(botToken string) (string, error) {
	botAPIURL := officialBotAPIURL

	if t.BotAPIURL != "" {
		if err := validateBotAPIURL(t.BotAPIURL); err != nil {
			return "", err
		}

		botAPIURL = t.BotAPIURL
	}

	return fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(botAPIURL, "/"), botToken), nil
}

// The URL is stored and returned in clear, so it must not be able to carry a secret of its own.
// A query or fragment would also break the path the bot token is appended to.
func validateBotAPIURL(rawBotAPIURL string) error {
	parsedBotAPIURL, err := url.Parse(rawBotAPIURL)
	if err != nil ||
		(parsedBotAPIURL.Scheme != "http" && parsedBotAPIURL.Scheme != "https") ||
		parsedBotAPIURL.Host == "" {
		return errors.New("bot API URL must be a valid http/https URL with a host")
	}

	if parsedBotAPIURL.User != nil {
		return errors.New("bot API URL must not contain a username or password")
	}

	// An empty query or fragment ("/telegram?") parses to empty fields, so the delimiters are checked
	// in the raw string. Outside user info, rejected above, they can only be those delimiters.
	if strings.ContainsAny(rawBotAPIURL, "?#") {
		return errors.New("bot API URL must not contain a query or fragment")
	}

	return nil
}

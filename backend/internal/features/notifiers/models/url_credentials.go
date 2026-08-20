package notifier_models

import (
	"errors"
	"net/url"
	"strings"
)

// Percent-encoding would mangle a literal "***" here; this marker survives url.String().
const redactedURLSegment = "redacted"

// For every webhook notifier we support, the URL itself is the credential, and it sits in the path
// where the logger's userinfo-based redaction cannot see it. Callers that know the URL is not a
// secret - Mattermost in bot mode - keep the path by not calling this.
func StripWebhookURLCredentials(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return redactedURLSegment
	}

	if parsed.User != nil {
		parsed.User = url.User(redactedURLSegment)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = redactPathSecrets(parsed.Path)

	return parsed.String()
}

// The transport error is what carries the credential off the machine: it is persisted to
// LastSendError, logged, and returned verbatim by the test-send endpoints. Rewriting the URL rather
// than unwrapping the *url.Error keeps the operation and host, so a failure stays diagnosable.
func ErrorWithoutWebhookURLCredentials(err error) error {
	var urlError *url.Error
	if !errors.As(err, &urlError) {
		return err
	}

	return &url.Error{
		Op:  urlError.Op,
		URL: StripWebhookURLCredentials(urlError.URL),
		Err: urlError.Err,
	}
}

// Which segment holds the secret varies per provider. The first one usually names the endpoint and
// is what makes a failure diagnosable, so it survives.
func redactPathSecrets(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return path
	}

	segments := strings.Split(trimmed, "/")
	for i := 1; i < len(segments); i++ {
		segments[i] = redactedURLSegment
	}

	segments[0] = redactFusedBotTokenSegment(segments[0])

	return "/" + strings.Join(segments, "/")
}

// Telegram is the one provider whose secret cannot be handled positionally: it fuses the "bot"
// prefix and the token into the endpoint segment itself.
func redactFusedBotTokenSegment(segment string) string {
	if !strings.HasPrefix(segment, "bot") || !strings.Contains(segment, ":") {
		return segment
	}

	return "bot" + redactedURLSegment
}

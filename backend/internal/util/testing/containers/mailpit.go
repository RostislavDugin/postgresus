package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	mailpitSmtpPort = "1025/tcp"
	mailpitHttpPort = "8025/tcp"
)

// MailpitEndpoint exposes both of Mailpit's ports: SMTP to submit mail and HTTP to assert via its
// message API what actually arrived.
type MailpitEndpoint struct {
	SMTP Endpoint
	HTTP Endpoint
}

// StartMailpit boots a Mailpit SMTP sink that accepts any auth over an insecure connection, so a
// test notifier can submit without TLS and real credentials.
func StartMailpit(t *testing.T) MailpitEndpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "axllent/mailpit:v1.20.5",
		ExposedPorts: []string{mailpitSmtpPort, mailpitHttpPort},
		Env: map[string]string{
			"MP_SMTP_AUTH_ACCEPT_ANY":     "true",
			"MP_SMTP_AUTH_ALLOW_INSECURE": "true",
		},
		WaitingFor: wait.ForListeningPort(mailpitSmtpPort).WithStartupTimeout(120 * time.Second),
	}

	handle := startContainer(t, req, mailpitSmtpPort)

	return MailpitEndpoint{
		SMTP: handle.Endpoint,
		HTTP: endpointOf(t, handle.Container, mailpitHttpPort),
	}
}

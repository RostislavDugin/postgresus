package email

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/testing/mailpit"
)

func Test_EmailSMTPSenderSendEmail_WhenSmtpServerAccepts_DeliversMessageToRecipient(t *testing.T) {
	mailpitEndpoint := containers.StartMailpit(t)
	mailpitClient := mailpit.NewClient(
		fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port),
	)

	sender := &EmailSMTPSender{
		logger:       logger.GetLogger(),
		smtpHost:     mailpitEndpoint.SMTP.Host,
		smtpPort:     mailpitEndpoint.SMTP.Port,
		smtpFrom:     "sender@databasus.local",
		isConfigured: true,
	}

	err := sender.SendEmail("recipient@databasus.local", "Password Reset Code", "<b>123456</b>")
	require.NoError(t, err)

	var delivered []mailpit.Message
	require.Eventually(t, func() bool {
		messages, fetchErr := mailpitClient.FetchMessages()
		if fetchErr != nil {
			return false
		}

		delivered = messages

		return len(messages) == 1
	}, 5*time.Second, 100*time.Millisecond, "Mailpit should receive exactly one message")

	assert.Equal(t, "Password Reset Code", delivered[0].Subject)
	require.Len(t, delivered[0].To, 1)
	assert.Equal(t, "recipient@databasus.local", delivered[0].To[0].Address)
}

func Test_EmailSMTPSenderImplicitTLS_WithUntrustedCert_FailsUnlessSkipVerify(t *testing.T) {
	host, port := startImplicitTLSGreetingServer(t, newSelfSignedTLSConfig(t))

	t.Run("verification on rejects the untrusted certificate", func(t *testing.T) {
		sender := &EmailSMTPSender{logger: logger.GetLogger(), smtpHost: host, smtpPort: port}

		_, _, dialErr := sender.createImplicitTLSClient()

		require.Error(t, dialErr)
		assert.Contains(t, dialErr.Error(), "certificate")
	})

	t.Run("skip verify accepts the untrusted certificate", func(t *testing.T) {
		sender := &EmailSMTPSender{
			logger:               logger.GetLogger(),
			smtpHost:             host,
			smtpPort:             port,
			isInsecureSkipVerify: true,
		}

		client, cleanup, dialErr := sender.createImplicitTLSClient()
		require.NoError(t, dialErr)
		defer cleanup()

		require.NotNil(t, client)
	})
}

// newSelfSignedTLSConfig builds a TLS config whose certificate is not signed by any trusted
// root, so a verifying client rejects it and a skip-verify client accepts it.
func newSelfSignedTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().UTC().Add(-time.Hour),
		NotAfter:     time.Now().UTC().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		IsCA:         true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	}
}

// startImplicitTLSGreetingServer accepts implicit-TLS connections and speaks just enough SMTP
// (greeting, EHLO, QUIT) for createImplicitTLSClient to connect and disconnect cleanly.
func startImplicitTLSGreetingServer(t *testing.T, tlsConfig *tls.Config) (host string, port int) {
	t.Helper()

	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}

			go serveMinimalSMTP(conn)
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)

	return addr.IP.String(), addr.Port
}

func serveMinimalSMTP(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte("220 test ESMTP\r\n")); err != nil {
		return
	}

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		if strings.HasPrefix(strings.ToUpper(line), "QUIT") {
			_, _ = conn.Write([]byte("221 Bye\r\n"))
			return
		}

		_, _ = conn.Write([]byte("250 OK\r\n"))
	}
}

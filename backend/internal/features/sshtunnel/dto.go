package sshtunnel

import (
	"log/slog"
	"net"
	"strconv"
	"time"

	"databasus-backend/internal/util/encryption"
)

type Endpoint struct {
	Host string
	Port int
}

func (e Endpoint) String() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

type OpenSpec struct {
	Config    Config
	Target    Endpoint
	Encryptor encryption.FieldEncryptor
	Logger    *slog.Logger

	// Zero or negative selects defaultHandshakeTimeout. The dialer bounds the TCP connect, but
	// nothing bounds the SSH handshake, so a bastion that accepts and then says nothing needs this.
	HandshakeTimeout time.Duration
}

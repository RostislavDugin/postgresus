package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials and the writable directory baked into the test atmoz/sftp container.
const (
	SftpUsername  = "testuser"
	SftpPassword  = "testpassword"
	SftpUploadDir = "upload"
)

const sftpPort = "22/tcp"

func StartSftp(t *testing.T) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "atmoz/sftp:latest",
		ExposedPorts: []string{sftpPort},
		Cmd:          []string{SftpUsername + ":" + SftpPassword + ":1001::" + SftpUploadDir},
		WaitingFor:   wait.ForListeningPort(sftpPort).WithStartupTimeout(120 * time.Second),
	}

	return start(t, req, sftpPort)
}

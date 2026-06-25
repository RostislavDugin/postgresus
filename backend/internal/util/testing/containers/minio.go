package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test MinIO container (the S3-compatible backend).
const (
	MinioRootUser     = "testuser"
	MinioRootPassword = "testpassword"
	MinioRegion       = "us-east-1"
)

const minioPort = "9000/tcp"

// StartMinio's container starts empty: the caller creates whatever bucket it needs against the
// returned endpoint, or for failure-path tests points at a missing one.
func StartMinio(t *testing.T) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{minioPort},
		Env: map[string]string{
			"MINIO_ROOT_USER":     MinioRootUser,
			"MINIO_ROOT_PASSWORD": MinioRootPassword,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort(minioPort).WithStartupTimeout(120 * time.Second),
	}

	return start(t, req, minioPort)
}

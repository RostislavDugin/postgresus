package containers

import (
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// Credentials baked into every test MariaDB container.
const (
	MariadbRootPassword = "rootpassword"
	MariadbUsername     = "testuser"
	MariadbPassword     = "testpassword"
	MariadbDatabase     = "testdb"
)

const mariadbPort = "3306/tcp"

func mariadbEnv() map[string]string {
	return map[string]string{
		"MARIADB_ROOT_PASSWORD": MariadbRootPassword,
		"MARIADB_DATABASE":      MariadbDatabase,
		"MARIADB_USER":          MariadbUsername,
		"MARIADB_PASSWORD":      MariadbPassword,
	}
}

// mariadbRequest builds the container request for image (e.g. "mariadb:10.11"). MariaDB runs the
// shared MySQL-family server flags (mysqlFamilyCmd) unchanged.
func mariadbRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{mariadbPort},
		Env:          mariadbEnv(),
		Cmd:          mysqlFamilyCmd(),
		Tmpfs:        map[string]string{"/var/lib/mysql": dataDirTmpfsOptions},
		WaitingFor:   mysqlFamilyReady(mariadbPort),
	}
}

// StartMariadb boots a MariaDB server from image (e.g. "mariadb:10.11").
func StartMariadb(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, mariadbRequest(image), mariadbPort)
}

// StartMariadbSSL boots a MariaDB server that rejects unencrypted connections. The image
// auto-generates its server certificates, so the test connects with tlsInsecure and no client cert.
func StartMariadbSSL(t *testing.T, image string) Endpoint {
	t.Helper()

	req := mariadbRequest(image)
	req.Cmd = append(req.Cmd, "--require-secure-transport=ON")

	return start(t, req, mariadbPort)
}

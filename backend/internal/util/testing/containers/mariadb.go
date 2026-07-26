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

func mariadbReadinessSpec(isSslRequired bool) mysqlFamilyReadinessSpec {
	return mysqlFamilyReadinessSpec{
		Port:          mariadbPort,
		RootPassword:  MariadbRootPassword,
		Database:      MariadbDatabase,
		IsSslRequired: isSslRequired,
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
		WaitingFor:   mysqlFamilyReady(mariadbReadinessSpec(false)),
	}
}

// StartMariadb boots a MariaDB server from image (e.g. "mariadb:10.11").
func StartMariadb(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, mariadbRequest(image), mariadbPort)
}

// StartMariadbSSL boots a MariaDB server that rejects unencrypted connections. MariaDB
// auto-generates its self-signed server certificates only from 11.4, so an older image leaves the
// server unreachable and burns the full readiness timeout. The test connects with tlsInsecure and
// no client cert.
func StartMariadbSSL(t *testing.T, image string) Endpoint {
	t.Helper()

	req := mariadbRequest(image)
	req.Cmd = append(req.Cmd, "--require-secure-transport=ON")
	req.WaitingFor = mysqlFamilyReady(mariadbReadinessSpec(true))

	return start(t, req, mariadbPort)
}

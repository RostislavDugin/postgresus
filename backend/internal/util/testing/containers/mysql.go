package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test MySQL container.
const (
	MysqlRootPassword = "rootpassword"
	MysqlUsername     = "testuser"
	MysqlPassword     = "testpassword"
	MysqlDatabase     = "testdb"
)

const mysqlPort = "3306/tcp"

// mysqlStartupTimeout is generous because go test -p=N boots many containers at once; the two-phase
// MySQL/MariaDB cold init (temp server for initdb, then a restart of the real one) runs under CPU
// contention and needs far more than its uncontended ~20s. A fast host returns as soon as the
// server is ready, so the ceiling is free there.
const mysqlStartupTimeout = 300 * time.Second

func mysqlEnv() map[string]string {
	return map[string]string{
		"MYSQL_ROOT_PASSWORD": MysqlRootPassword,
		"MYSQL_DATABASE":      MysqlDatabase,
		"MYSQL_USER":          MysqlUsername,
		"MYSQL_PASSWORD":      MysqlPassword,
	}
}

// mysqlFamilyReady waits for the network-listening server on port. The entrypoint starts a
// temporary socket-only server for initdb and then restarts the real one, so "ready for
// connections" is logged twice; the second occurrence is the real server. MariaDB shares this
// behaviour, so it passes its own port.
func mysqlFamilyReady(port string) wait.Strategy {
	return wait.ForAll(
		wait.ForLog("ready for connections").WithOccurrence(2).WithStartupTimeout(mysqlStartupTimeout),
		wait.ForListeningPort(port),
	)
}

// mysqlFamilyCmd returns the server flags shared by MySQL and MariaDB: the utf8mb4 charset
// defaults, plus durability-off flags (no fsync, doublewrite or binary log) that are safe for
// throwaway tmpfs containers and make the cold init and the e2e restore RAM-fast.
func mysqlFamilyCmd() []string {
	return []string{
		"--character-set-server=utf8mb4",
		"--collation-server=utf8mb4_unicode_ci",
		"--innodb-flush-log-at-trx-commit=0",
		"--innodb-doublewrite=0",
		"--sync-binlog=0",
		"--skip-log-bin",
	}
}

// mysqlCmd appends the native password plugin for mysql:8.0 (removed in 8.4+) to the shared family
// flags; later versions use the family flags unchanged.
func mysqlCmd(image string) []string {
	cmd := mysqlFamilyCmd()

	if image == "mysql:8.0" {
		cmd = append(cmd, "--default-authentication-plugin=mysql_native_password")
	}

	return cmd
}

// mysqlRequest builds the container request for image (e.g. "mysql:8.0").
func mysqlRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{mysqlPort},
		Env:          mysqlEnv(),
		Cmd:          mysqlCmd(image),
		Tmpfs:        map[string]string{"/var/lib/mysql": dataDirTmpfsOptions},
		WaitingFor:   mysqlFamilyReady(mysqlPort),
	}
}

// StartMysql boots a MySQL server from image (e.g. "mysql:8.0").
func StartMysql(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, mysqlRequest(image), mysqlPort)
}

// StartMysqlSSL boots a MySQL server that rejects unencrypted connections. The image auto-generates
// its server certificates, so the test connects with tlsInsecure and no client cert.
func StartMysqlSSL(t *testing.T, image string) Endpoint {
	t.Helper()

	req := mysqlRequest(image)
	req.Cmd = append(req.Cmd, "--require_secure_transport=ON")

	return start(t, req, mysqlPort)
}

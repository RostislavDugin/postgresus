package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test MongoDB container.
const (
	MongodbUsername     = "root"
	MongodbPassword     = "rootpassword"
	MongodbDatabase     = "testdb"
	MongodbAuthDatabase = "admin"
)

const mongodbPort = "27017/tcp"

// mongodbStartupTimeout is generous because go test -p=N starts many mongod containers at once;
// under CPU contention a cold boot runs much longer than its uncontended time. A fast host returns
// as soon as mongod is listening, so the ceiling is free there.
const mongodbStartupTimeout = 240 * time.Second

func mongodbEnv() map[string]string {
	return map[string]string{
		"MONGO_INITDB_ROOT_USERNAME": MongodbUsername,
		"MONGO_INITDB_ROOT_PASSWORD": MongodbPassword,
		"MONGO_INITDB_DATABASE":      MongodbDatabase,
	}
}

// mongodbRequest builds the container request for an auth-enabled mongod from image
// (e.g. "mongo:7.0").
func mongodbRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{mongodbPort},
		Env:          mongodbEnv(),
		Cmd:          []string{"mongod", "--auth"},
		Tmpfs:        map[string]string{"/data/db": dataDirTmpfsOptions},
		WaitingFor:   wait.ForListeningPort(mongodbPort).WithStartupTimeout(mongodbStartupTimeout),
	}
}

// StartMongodb boots an auth-enabled mongod from image (e.g. "mongo:7.0").
func StartMongodb(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, mongodbRequest(image), mongodbPort)
}

// StartMongodbSSL boots a requireTLS mongod, copying the server key/cert from pemPath and crtPath
// into the container. tlsAllowConnectionsWithoutCertificates lets clients connect without a client
// cert (the SSL test uses tlsInsecure).
func StartMongodbSSL(t *testing.T, pemPath, crtPath string) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:8.2.3-noble",
		ExposedPorts: []string{mongodbPort},
		Env:          mongodbEnv(),
		Files: []testcontainers.ContainerFile{
			{HostFilePath: pemPath, ContainerFilePath: "/etc/ssl-test/server.pem", FileMode: 0o644},
			{HostFilePath: crtPath, ContainerFilePath: "/etc/ssl-test/server.crt", FileMode: 0o644},
		},
		Cmd: []string{
			"mongod", "--auth",
			"--tlsMode", "requireTLS",
			"--tlsCertificateKeyFile", "/etc/ssl-test/server.pem",
			"--tlsCAFile", "/etc/ssl-test/server.crt",
			"--tlsAllowConnectionsWithoutCertificates",
		},
		WaitingFor: wait.ForListeningPort(mongodbPort).WithStartupTimeout(mongodbStartupTimeout),
	}

	return start(t, req, mongodbPort)
}

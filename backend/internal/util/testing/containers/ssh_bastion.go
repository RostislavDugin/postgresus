package containers

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into the test bastion image.
const (
	SshBastionUsername = "testuser"
	SshBastionPassword = "testpassword"
)

const sshBastionPort = "22/tcp"

// The image is built from contextDir rather than pulled because the stock openssh images disagree
// on whether AllowTcpForwarding defaults to yes, and the baked-in authorized_keys has to match the
// private key the test signs with.
func StartSshBastion(t *testing.T) Endpoint {
	t.Helper()

	return start(t, sshBastionRequest(t), sshBastionPort)
}

// Port 22 stays published so the test can dial the bastion; the network membership is what lets the
// bastion in turn reach containers that publish nothing.
func StartSshBastionOnNetwork(t *testing.T, networkName string) Endpoint {
	t.Helper()

	req := sshBastionRequest(t)
	req.Networks = []string{networkName}

	return start(t, req, sshBastionPort)
}

// Resolved from this file rather than the working directory: callers live in other packages, and
// testcontainers resolves a relative context against the CWD. Tests that need the key material
// baked into the image read it from here too.
func GetSshBastionTestdataDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, isResolved := runtime.Caller(0)
	if !isResolved {
		t.Fatal("failed to resolve the ssh bastion testdata directory")
	}

	return filepath.Join(filepath.Dir(thisFile), "testdata", "ssh_bastion")
}

// Database is the in-network address, reachable only through Bastion.
type BastionedDatabase struct {
	Bastion  Endpoint
	Database Endpoint
}

// One alias per engine so a run that boots several topologies stays readable in docker inspect.
const (
	bastionedPostgresAlias = "postgres-behind-bastion"
	bastionedMysqlAlias    = "mysql-behind-bastion"
	bastionedMariadbAlias  = "mariadb-behind-bastion"
	bastionedMongodbAlias  = "mongodb-behind-bastion"
)

// The database publishes no ports and the bastion does, so a tunnel is the only way in. Tests
// asserting that traffic really goes through the tunnel need that: a direct route would keep them
// green after the tunnel stopped being used.
func StartPostgresBehindSshBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	return BastionedDatabase{
		Bastion: StartSshBastionOnNetwork(t, networkName),
		Database: StartPostgresOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedPostgresAlias,
			},
		}),
	}
}

func StartMysqlBehindSshBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	return BastionedDatabase{
		Bastion: StartSshBastionOnNetwork(t, networkName),
		Database: StartMysqlOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedMysqlAlias,
			},
		}),
	}
}

func StartMariadbBehindSshBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	return BastionedDatabase{
		Bastion: StartSshBastionOnNetwork(t, networkName),
		Database: StartMariadbOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedMariadbAlias,
			},
		}),
	}
}

func StartMongodbBehindSshBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	return BastionedDatabase{
		Bastion: StartSshBastionOnNetwork(t, networkName),
		Database: StartMongodbOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedMongodbAlias,
			},
		}),
	}
}

func sshBastionRequest(t *testing.T) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    GetSshBastionTestdataDir(t),
			Dockerfile: "Dockerfile",
			Repo:       "databasus-test-ssh-bastion",
			Tag:        "latest",
			KeepImage:  true,
		},
		ExposedPorts: []string{sshBastionPort},
		WaitingFor:   wait.ForListeningPort(sshBastionPort).WithStartupTimeout(120 * time.Second),
	}
}

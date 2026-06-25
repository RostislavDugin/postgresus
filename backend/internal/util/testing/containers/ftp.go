package containers

import (
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into the test pure-ftpd container.
const (
	FtpUsername = "testuser"
	FtpPassword = "testpassword"
)

const ftpControlPort = "21/tcp"

// pure-ftpd hands out a port from this range on EPSV, and jlaffaye/ftp (EPSV by default) then dials
// the control host at that exact port. So the range must be published 1:1 (container port == host
// port) instead of at testcontainers' usual random host port, or the data connection lands nowhere.
// Only the storages package uses FTP and its sub-tests run one container at a time, so the fixed
// host range never collides across the suite.
const (
	ftpPassivePortMin = 30000
	ftpPassivePortMax = 30009
)

// StartFtp binds the passive port range 1:1 (see the ftpPassivePort constant comment); the control
// port stays dynamic and is read back as the endpoint.
func StartFtp(t *testing.T) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "stilliard/pure-ftpd:latest",
		ExposedPorts: ftpExposedPorts(),
		Env: map[string]string{
			"PUBLICHOST":        "localhost",
			"FTP_USER_NAME":     FtpUsername,
			"FTP_USER_PASS":     FtpPassword,
			"FTP_USER_HOME":     "/home/ftpusers/" + FtpUsername,
			"FTP_PASSIVE_PORTS": strconv.Itoa(ftpPassivePortMin) + ":" + strconv.Itoa(ftpPassivePortMax),
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = withPassivePortRange(hc.PortBindings)
		},
		WaitingFor: wait.ForListeningPort(ftpControlPort).WithStartupTimeout(120 * time.Second),
	}

	return start(t, req, ftpControlPort)
}

// ftpExposedPorts lists the control port plus every passive port. testcontainers drops any host
// binding whose port is not also exposed (its mergePortBindings filters to the exposed set), so the
// passive range must appear here for the fixed 1:1 bindings in withPassivePortRange to survive.
func ftpExposedPorts() []string {
	ports := []string{ftpControlPort}
	for passivePort := ftpPassivePortMin; passivePort <= ftpPassivePortMax; passivePort++ {
		ports = append(ports, strconv.Itoa(passivePort)+"/tcp")
	}

	return ports
}

// withPassivePortRange adds a fixed host-port binding for each FTP passive port, preserving the
// dynamic control-port binding testcontainers populates from ExposedPorts.
func withPassivePortRange(bindings network.PortMap) network.PortMap {
	if bindings == nil {
		bindings = network.PortMap{}
	}

	for passivePort := ftpPassivePortMin; passivePort <= ftpPassivePortMax; passivePort++ {
		hostPort := strconv.Itoa(passivePort)
		bindings[network.MustParsePort(hostPort+"/tcp")] = []network.PortBinding{{HostPort: hostPort}}
	}

	return bindings
}

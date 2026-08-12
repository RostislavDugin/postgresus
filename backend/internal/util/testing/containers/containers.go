// Package containers spins up throwaway database containers for integration tests via
// testcontainers-go. Each Start* helper boots one container, waits until it is ready and registers
// a t.Cleanup that terminates it, so a container lives only for the duration of the test that
// created it. Each engine gets its own file; this file holds the shared plumbing.
package containers

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

// dataDirTmpfsOptions mounts a container's data directory on tmpfs (RAM) instead of the overlay
// filesystem, so the fsync-heavy cold init of the SQL engines is RAM-fast. The size is pinned
// because Docker's tmpfs default is half the host RAM, which is unsafe to reserve per container
// under go test -p=N; 512m dwarfs every test fixture and tmpfs only consumes the bytes written.
const dataDirTmpfsOptions = "rw,size=512m"

const loopbackIP = "127.0.0.1"

// Endpoint is the reachable address of a started container's primary port.
type Endpoint struct {
	Host string
	Port int
}

// ContainerHandle is an Endpoint plus the live container, for tests that must Exec or copy files
// into the container (e.g. the physical restore target reconstructs a cluster in place) rather than
// only dial its port.
type ContainerHandle struct {
	Endpoint
	Container testcontainers.Container
}

// For tests that need one container reachable only from another rather than from the host.
func StartNetwork(t *testing.T) string {
	t.Helper()

	testNetwork, err := network.New(context.Background())
	if err != nil {
		t.Fatalf("failed to create test network: %v", err)
	}

	t.Cleanup(func() {
		if err := testNetwork.Remove(context.Background()); err != nil {
			t.Logf("failed to remove test network: %v", err)
		}
	})

	return testNetwork.Name
}

// Publishing no ports leaves the server unreachable from the host, dialable only by another
// container on the network. The Endpoint each Start*OnNetwork returns is that in-network
// address, which is what an SSH bastion on the same network resolves.
type OnNetworkSpec struct {
	Image     string
	Placement NetworkPlacement
}

type NetworkPlacement struct {
	NetworkName string
	Alias       string
}

// The engine port consts are testcontainers-shaped ("5432/tcp"); an unpublished container is dialed
// by that raw number rather than by a mapped one.
func getPortNumber(exposedPort string) int {
	number, _, _ := strings.Cut(exposedPort, "/")

	port, err := strconv.Atoi(number)
	if err != nil {
		panic("malformed exposed port " + exposedPort)
	}

	return port
}

func startUnpublished(t *testing.T, req testcontainers.ContainerRequest, placement NetworkPlacement) {
	t.Helper()

	req.ExposedPorts = nil
	req.Networks = []string{placement.NetworkName}
	req.NetworkAliases = map[string][]string{placement.NetworkName: {placement.Alias}}

	container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start %s container: %v", req.Image, err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate %s container: %v", req.Image, err)
		}
	})
}

func start(t *testing.T, req testcontainers.ContainerRequest, mappedPort string) Endpoint {
	t.Helper()

	return startContainer(t, req, mappedPort).Endpoint
}

func startContainer(t *testing.T, req testcontainers.ContainerRequest, mappedPort string) ContainerHandle {
	t.Helper()

	container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start %s container: %v", req.Image, err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate %s container: %v", req.Image, err)
		}
	})

	return ContainerHandle{Endpoint: endpointOf(t, container, mappedPort), Container: container}
}

// The MySQL client resolves the name "localhost" to a unix socket rather than TCP, and the server
// lives in a container where no such socket exists.
func endpointOf(t *testing.T, container testcontainers.Container, mappedPort string) Endpoint {
	t.Helper()

	ctx := context.Background()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	if host == "localhost" {
		host = loopbackIP
	}

	port, err := container.MappedPort(ctx, mappedPort)
	if err != nil {
		t.Fatalf("failed to get container mapped port: %v", err)
	}

	return Endpoint{Host: host, Port: int(port.Num())}
}

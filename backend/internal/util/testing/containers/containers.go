// Package containers spins up throwaway database containers for integration tests via
// testcontainers-go. Each Start* helper boots one container, waits until it is ready and registers
// a t.Cleanup that terminates it, so a container lives only for the duration of the test that
// created it. Each engine gets its own file; this file holds the shared plumbing.
package containers

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// dataDirTmpfsOptions mounts a container's data directory on tmpfs (RAM) instead of the overlay
// filesystem, so the fsync-heavy cold init of the SQL engines is RAM-fast. The size is pinned
// because Docker's tmpfs default is half the host RAM, which is unsafe to reserve per container
// under go test -p=N; 512m dwarfs every test fixture and tmpfs only consumes the bytes written.
const dataDirTmpfsOptions = "rw,size=512m"

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

// endpointOf honours TESTCONTAINERS_HOST_OVERRIDE, which CI sets to reach containers published on
// the Docker host bridge.
func endpointOf(t *testing.T, container testcontainers.Container, mappedPort string) Endpoint {
	t.Helper()

	ctx := context.Background()

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, mappedPort)
	if err != nil {
		t.Fatalf("failed to get container mapped port: %v", err)
	}

	return Endpoint{Host: host, Port: int(port.Num())}
}

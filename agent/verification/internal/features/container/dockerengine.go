// Package container owns the ephemeral Postgres container lifecycle. Every
// Docker-SDK call is confined to this file; the Manager builds a hardened
// SpawnSpec (unit-tested) that this file translates 1:1 to Docker.
package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"

	"databasus-verification-agent/internal/features/restore"
)

const (
	// LabelVerificationID / LabelAgentID tag every container and network so the
	// startup purge can find and remove this agent's leftovers by agent ID
	LabelVerificationID = "databasus.verification.verification_id"
	LabelAgentID        = "databasus.verification.agent_id"

	containerPidsLimit = 512

	// pgInternalPort is the in-container Postgres port; it is published to an
	// ephemeral 127.0.0.1 host port for the verifier.
	pgInternalPort = 5432
)

// pgPublishedPort is the parsed network.Port key reused for ExposedPorts and
// PortBindings; ParsePort is the only constructor in the new SDK.
var pgPublishedPort = mustParsePort(strconv.Itoa(pgInternalPort) + "/tcp")

func mustParsePort(s string) network.Port {
	p, err := network.ParsePort(s)
	if err != nil {
		panic(fmt.Sprintf("parse port %q: %v", s, err))
	}

	return p
}

type dockerEngine struct {
	cli *client.Client
}

func NewDockerEngine() (*dockerEngine, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &dockerEngine{cli: cli}, nil
}

func (e *dockerEngine) Ping(ctx context.Context) error {
	_, err := e.cli.Ping(ctx, client.PingOptions{})
	return err
}

func (e *dockerEngine) GetHostCapabilities(ctx context.Context) (HostCapabilities, error) {
	result, err := e.cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return HostCapabilities{}, err
	}

	return toHostCapabilities(result.Info), nil
}

// toHostCapabilities requires both CFS flags because that is the daemon's own
// condition for accepting NanoCPUs; older daemons track them separately.
func toHostCapabilities(info system.Info) HostCapabilities {
	isUserNSRemapEnabled := slices.ContainsFunc(info.SecurityOptions, func(opt string) bool {
		return strings.Contains(opt, "name=userns")
	})

	return HostCapabilities{
		IsUserNSRemapEnabled: isUserNSRemapEnabled,
		IsCpuLimitSupported:  info.CPUCfsPeriod && info.CPUCfsQuota,
	}
}

func (e *dockerEngine) EnsureImage(ctx context.Context, ref string) error {
	if _, err := e.cli.ImageInspect(ctx, ref); err == nil {
		return nil
	}

	reader, err := e.cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("image pull: %w", err)
	}
	defer func() { _ = reader.Close() }()

	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("drain image pull: %w", err)
	}

	return nil
}

func (e *dockerEngine) CreateNetwork(
	ctx context.Context, name string, labels map[string]string,
) (string, error) {
	// Each job gets its own user-defined bridge — that isolates jobs from each
	// other. We deliberately do NOT set Internal:true: Docker silently disables
	// port publishing on internal networks, and the agent-process verifier can
	// only reach the restored DB via the published host port (verifier package
	// doc). Outbound from PG to the host network is therefore not blocked at
	// the docker layer; rely on the host firewall + the other hardening
	// controls (cap-drop, no-new-privs, pids/memory, ephemeral, userns).
	resp, err := e.cli.NetworkCreate(ctx, name, client.NetworkCreateOptions{
		Driver: "bridge",
		Labels: labels,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (e *dockerEngine) RemoveNetwork(ctx context.Context, networkID string) error {
	_, err := e.cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
	return err
}

func (e *dockerEngine) CreateContainer(ctx context.Context, spec SpawnSpec) (string, error) {
	securityOpt := []string{}
	if spec.NoNewPrivileges {
		securityOpt = append(securityOpt, "no-new-privileges")
	}

	capDrop := []string{}
	if spec.CapDropAll {
		capDrop = append(capDrop, "ALL")
	}

	var pidsLimit *int64
	if spec.PidsLimit > 0 {
		limit := spec.PidsLimit
		pidsLimit = &limit
	}

	hostConfig := &container.HostConfig{
		SecurityOpt:    securityOpt,
		CapDrop:        capDrop,
		CapAdd:         spec.CapAdd,
		ReadonlyRootfs: spec.ReadonlyRootfs,
		NetworkMode:    container.NetworkMode(spec.NetworkID),
		PortBindings: network.PortMap{
			pgPublishedPort: []network.PortBinding{
				{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "0"},
			},
		},
		Resources: container.Resources{
			NanoCPUs:  spec.NanoCPUs,
			Memory:    spec.MemoryBytes,
			PidsLimit: pidsLimit,
		},
	}

	resp, err := e.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:        spec.Image,
			Cmd:          spec.Cmd,
			Env:          spec.Env,
			Labels:       spec.Labels,
			ExposedPorts: network.PortSet{pgPublishedPort: struct{}{}},
		},
		HostConfig: hostConfig,
		Name:       spec.Name,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (e *dockerEngine) StartContainer(ctx context.Context, containerID string) error {
	_, err := e.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	return err
}

func (e *dockerEngine) HostPort(
	ctx context.Context, containerID, containerPort string,
) (int, error) {
	insp, err := e.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return 0, err
	}

	requestedPort, err := network.ParsePort(containerPort)
	if err != nil {
		return 0, fmt.Errorf("parse container port %q: %w", containerPort, err)
	}

	bindings, ok := insp.Container.NetworkSettings.Ports[requestedPort]
	if !ok || len(bindings) == 0 {
		return 0, fmt.Errorf("container port %s is not published", containerPort)
	}

	return strconv.Atoi(bindings[0].HostPort)
}

func (e *dockerEngine) InspectSecurity(
	ctx context.Context, containerID string,
) (SecurityState, error) {
	insp, err := e.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return SecurityState{}, err
	}

	state := SecurityState{
		ReadonlyRootfs: insp.Container.HostConfig.ReadonlyRootfs,
		// Only flag mounts the AGENT requested. insp.Mounts (runtime) includes
		// image-declared anonymous volumes (postgres:16 declares VOLUME
		// /var/lib/postgresql/data for PGDATA) that we explicitly want; only
		// host bind mounts from our spec are a hardening regression.
		HasHostBinds: len(insp.Container.HostConfig.Binds) > 0 || len(insp.Container.HostConfig.Mounts) > 0,
	}

	for _, opt := range insp.Container.HostConfig.SecurityOpt {
		if strings.Contains(opt, "no-new-privileges") {
			state.NoNewPrivileges = true
		}
	}

	for _, cap := range insp.Container.HostConfig.CapDrop {
		if cap == "ALL" {
			state.CapDropAll = true
		}
	}

	return state, nil
}

func (e *dockerEngine) Exec(
	ctx context.Context,
	containerID string,
	cmd []string,
	stdin io.Reader,
	env []string,
) (restore.ExecResult, error) {
	created, err := e.cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          cmd,
		Env:          env,
		AttachStdin:  stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec create: %w", err)
	}

	attached, err := e.cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec attach: %w", err)
	}
	defer attached.Close()

	// The hijacked conn's Read/Write are not ctx-aware: a container that wedges
	// while consuming stdin (or producing output) would otherwise hang Exec
	// indefinitely, defeating jobCtx/heartbeat-abort. Closing it on cancellation
	// unblocks the in-flight copy; the deferred Close above is then a harmless
	// second close.
	stopOnCancel := context.AfterFunc(ctx, func() { attached.Close() })
	defer stopOnCancel()

	if stdin != nil {
		copyErr := make(chan error, 1)
		go func() {
			_, cErr := io.Copy(attached.Conn, stdin)
			_ = attached.CloseWrite()
			copyErr <- cErr
		}()

		if cErr := <-copyErr; cErr != nil {
			return restore.ExecResult{}, fmt.Errorf("exec stdin copy: %w", cErr)
		}
	}

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attached.Reader); err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec stream copy: %w", err)
	}

	inspect, err := e.cli.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec inspect: %w", err)
	}

	return restore.ExecResult{
		ExitCode: inspect.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func (e *dockerEngine) RemoveContainer(ctx context.Context, containerID string) error {
	_, err := e.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	return err
}

func (e *dockerEngine) ListManaged(
	ctx context.Context, agentID string,
) ([]ManagedContainer, error) {
	listed, err := e.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", LabelAgentID+"="+agentID),
	})
	if err != nil {
		return nil, err
	}

	managed := make([]ManagedContainer, 0, len(listed.Items))
	for _, s := range listed.Items {
		mc := ManagedContainer{
			ID:             s.ID,
			VerificationID: s.Labels[LabelVerificationID],
			CreatedUnix:    s.Created,
		}

		if s.NetworkSettings != nil {
			for _, endpoint := range s.NetworkSettings.Networks {
				if endpoint.NetworkID != "" {
					mc.NetworkID = endpoint.NetworkID
					break
				}
			}
		}

		managed = append(managed, mc)
	}

	return managed, nil
}

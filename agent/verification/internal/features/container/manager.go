package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"databasus-verification-agent/internal/features/dbconn"
)

const (
	containerReadyTimeout = 60 * time.Second
	readyPollInterval     = 1 * time.Second

	// stockEngineImageRepo is the official Postgres image; the job's major version
	// (from the backend assignment) is appended as the tag. It is the fallback
	// when the configured (extension-bundling) repo can't be pulled.
	stockEngineImageRepo = "postgres"

	// timescaleImageRepo carries the timescaledb extension; the tag is
	// "<extversion>-pg<major>" so the engine matches the dump exactly
	// (pg_restore cannot cross extension versions).
	timescaleImageRepo = "timescale/timescaledb"
)

// minimalCaps are the only capabilities added back after CapDrop ALL — exactly
// what the official entrypoint needs for initdb + the gosu privilege drop, not
// a blanket keep.
var minimalCaps = []string{"CHOWN", "DAC_OVERRIDE", "FOWNER", "SETGID", "SETUID"}

// restoreTunedPostgresCmd runs the restore container's Postgres with crash
// safety traded for a smaller on-disk footprint. The container is destroyed
// after verification, so durability is irrelevant: fsync=off and
// full_page_writes=off cut WAL volume, faster checkpoints recycle pg_wal
// sooner. max_wal_senders=0 is mandatory under wal_level=minimal — Postgres
// refuses to start otherwise. The first arg stays "postgres" so the official
// image entrypoint still runs initdb.
var restoreTunedPostgresCmd = []string{
	"postgres",
	"-c", "fsync=off",
	"-c", "full_page_writes=off",
	"-c", "synchronous_commit=off",
	"-c", "wal_level=minimal",
	"-c", "max_wal_senders=0",
	"-c", "max_wal_size=1GB",
}

type Manager struct {
	engine          *dockerEngine
	agentID         string
	engineImageRepo string
	log             *slog.Logger
}

func NewManager(
	engine *dockerEngine,
	agentID string,
	engineImageRepo string,
	log *slog.Logger,
) *Manager {
	return &Manager{
		engine:          engine,
		agentID:         agentID,
		engineImageRepo: engineImageRepo,
		log:             log,
	}
}

func (m *Manager) StartupSelfCheck(ctx context.Context) error {
	if err := m.engine.Ping(ctx); err != nil {
		return fmt.Errorf("docker daemon unreachable: %w", err)
	}

	m.log.Info("docker daemon is reachable")

	remapOn, err := m.engine.UserNSRemapEnabled(ctx)
	if err != nil {
		m.log.Warn("could not determine user-namespace remap state", "error", err)
	} else if !remapOn {
		m.log.Warn(
			"docker user-namespace remapping is OFF — the strongest control " +
				"against a container escape is absent; enable userns-remap on the " +
				"verification host")
	}

	return nil
}

// PurgeContainers removes every container+network this agent owns so each
// process starts from a clean slate. Call once at startup, after
// StartupSelfCheck (Docker must be reachable) and before any job is claimed.
// See purgeAgentContainers for why blanket removal is safe under one agent.
func (m *Manager) PurgeContainers(ctx context.Context) {
	purgeAgentContainers(ctx, m.engine, m.agentID, m.log)
}

// Spawn's every failure is pre-pg_restore — the runner reports it with no exit
// code (retryable AgentSetupFailed), never BackupRejected.
func (m *Manager) Spawn(jobCtx context.Context, req SpawnRequest) (*PostgresContainer, error) {
	image := imageForJob(req, m.engineImageRepo)

	if err := m.engine.EnsureImage(jobCtx, image); err != nil {
		fallbackImage, canFallBack := stockFallbackImage(req, image)
		if !canFallBack {
			return nil, fmt.Errorf("ensure image: %w", err)
		}

		// The extension-bundling image is unreachable (registry down, air-gapped,
		// not yet published). Fall back to the stock Postgres image; missing
		// extensions then surface as ignored pg_restore errors the runner tolerates.
		m.log.Warn("bundled verification image unavailable; falling back to stock postgres image",
			"bundled_image", image, "stock_image", fallbackImage, "error", err)

		image = fallbackImage

		if err := m.engine.EnsureImage(jobCtx, image); err != nil {
			return nil, fmt.Errorf("ensure stock image: %w", err)
		}
	}

	password, err := randomPassword()
	if err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}

	// LabelAgentID is load-bearing for the startup purge: ListManaged filters
	// solely on it, so a container spawned without this label can never be
	// purged and leaks until the host is cleaned by hand.
	labels := map[string]string{
		LabelAgentID:        m.agentID,
		LabelVerificationID: req.VerificationID.String(),
	}

	networkID, err := m.engine.CreateNetwork(
		jobCtx, "databasus-verif-"+req.VerificationID.String(), labels)
	if err != nil {
		return nil, fmt.Errorf("create isolated network: %w", err)
	}

	spec := m.buildSpec(spawnPlan{
		verificationID: req.VerificationID,
		image:          image,
		password:       password,
		cpuPerJob:      req.CPUPerJob,
		ramMbPerJob:    req.RAMMbPerJob,
		networkID:      networkID,
		labels:         labels,
	})

	containerID, err := m.engine.CreateContainer(jobCtx, spec)
	if err != nil {
		_ = m.engine.RemoveNetwork(jobCtx, networkID)
		return nil, fmt.Errorf("create container: %w", err)
	}

	c := &PostgresContainer{engine: m.engine, id: containerID, networkID: networkID, password: password, log: m.log}

	if err := m.engine.StartContainer(jobCtx, containerID); err != nil {
		_ = c.Terminate(jobCtx)
		return nil, fmt.Errorf("start container: %w", err)
	}

	hostPort, err := m.engine.HostPort(jobCtx, containerID, strconv.Itoa(pgInternalPort)+"/tcp")
	if err != nil {
		_ = c.Terminate(jobCtx)
		return nil, fmt.Errorf("resolve published port: %w", err)
	}

	c.hostPort = hostPort

	if err := waitForReady(jobCtx, c.GetVerifierConn()); err != nil {
		_ = c.Terminate(jobCtx)
		return nil, fmt.Errorf("container never became ready: %w", err)
	}

	if err := m.assertSecurity(jobCtx, containerID); err != nil {
		_ = c.Terminate(jobCtx)
		return nil, err
	}

	return c, nil
}

func (m *Manager) buildSpec(plan spawnPlan) SpawnSpec {
	return SpawnSpec{
		Name:        "databasus-verif-" + plan.verificationID.String(),
		Image:       plan.image,
		Cmd:         restoreTunedPostgresCmd,
		Env:         []string{"POSTGRES_PASSWORD=" + plan.password},
		Labels:      plan.labels,
		NanoCPUs:    int64(plan.cpuPerJob) * 1_000_000_000,
		MemoryBytes: int64(plan.ramMbPerJob) * 1024 * 1024,
		PidsLimit:   containerPidsLimit,
		NetworkID:   plan.networkID,

		NoNewPrivileges: true,
		CapDropAll:      true,
		CapAdd:          minimalCaps,
		// rootfs is writable because the official image needs a writable PGDATA;
		// the per-job disk budget is enforced by the agent's disk watcher, not a
		// Docker cap. The dominant controls (cap-drop, no-new-privs, pids,
		// memory, userns, ephemeral lifecycle) remain in force. The job network
		// is a per-job user-defined bridge (job isolation) but NOT --internal:
		// see CreateNetwork in dockerengine.go for why.
		ReadonlyRootfs: false,
	}
}

func (m *Manager) assertSecurity(ctx context.Context, containerID string) error {
	state, err := m.engine.InspectSecurity(ctx, containerID)
	if err != nil {
		return fmt.Errorf("inspect security state: %w", err)
	}

	switch {
	case !state.NoNewPrivileges:
		return fmt.Errorf("hardening regression: no-new-privileges not applied")
	case !state.CapDropAll:
		return fmt.Errorf("hardening regression: CapDrop ALL not applied")
	case state.HasHostBinds:
		return fmt.Errorf("hardening regression: container has host bind mounts")
	}

	return nil
}

func waitForReady(ctx context.Context, conn dbconn.Conn) error {
	readyCtx, cancel := context.WithTimeout(ctx, containerReadyTimeout)
	defer cancel()

	ticker := time.NewTicker(readyPollInterval)
	defer ticker.Stop()

	for {
		if pingOnce(readyCtx, conn) {
			return nil
		}

		select {
		case <-readyCtx.Done():
			return readyCtx.Err()
		case <-ticker.C:
		}
	}
}

func pingOnce(ctx context.Context, conn dbconn.Conn) bool {
	pingCtx, cancel := context.WithTimeout(ctx, readyPollInterval)
	defer cancel()

	pgConn, err := pgx.Connect(pingCtx, conn.DSN())
	if err != nil {
		return false
	}
	defer func() { _ = pgConn.Close(pingCtx) }()

	return pgConn.Ping(pingCtx) == nil
}

func imageForJob(req SpawnRequest, engineImageRepo string) string {
	if req.TimescaledbVersion != "" {
		return fmt.Sprintf("%s:%s-pg%s", timescaleImageRepo, req.TimescaledbVersion, req.PgMajor)
	}

	return imageForMajor(req.PgMajor, engineImageRepo)
}

func imageForMajor(pgMajor, engineImageRepo string) string {
	return engineImageRepo + ":" + pgMajor
}

// stockFallbackImage decides whether a failed pull of the configured image may
// retry against the official Postgres image. Timescale jobs need their exact
// version-matched image (pg_restore cannot cross extension versions), so they
// never fall back; nor does a job already targeting the stock repo.
func stockFallbackImage(req SpawnRequest, attemptedImage string) (string, bool) {
	if req.TimescaledbVersion != "" {
		return "", false
	}

	stockImage := imageForMajor(req.PgMajor, stockEngineImageRepo)
	if stockImage == attemptedImage {
		return "", false
	}

	return stockImage, true
}

func randomPassword() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

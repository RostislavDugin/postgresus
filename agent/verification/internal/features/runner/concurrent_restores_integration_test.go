package runner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-verification-agent/internal/features/api"
	"databasus-verification-agent/internal/features/dbconn"
	"databasus-verification-agent/internal/features/heartbeat"
	"databasus-verification-agent/internal/features/restore"
	"databasus-verification-agent/internal/features/verifier"
	"databasus-verification-agent/internal/testutil"
)

type concurrentRestoreBackend struct {
	claims   []*api.JobAssignment
	claimIdx atomic.Int32

	mu      sync.Mutex
	reports map[uuid.UUID]api.VerificationStatus
}

func newConcurrentRestoreBackend(claims []*api.JobAssignment) (*concurrentRestoreBackend, *httptest.Server) {
	backend := &concurrentRestoreBackend{
		claims:  claims,
		reports: make(map[uuid.UUID]api.VerificationStatus),
	}

	mux := http.NewServeMux()

	mux.HandleFunc(
		"POST /api/v1/agent/verifications/{agentId}/claim",
		func(w http.ResponseWriter, _ *http.Request) {
			i := int(backend.claimIdx.Add(1)) - 1
			if i >= len(backend.claims) {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(backend.claims[i])
		},
	)

	mux.HandleFunc(
		"GET /api/v1/agent/verifications/{agentId}/{id}/backup-stream",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ARCHIVE"))
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verifications/{agentId}/{id}/report",
		func(w http.ResponseWriter, req *http.Request) {
			verificationID, err := uuid.Parse(req.PathValue("id"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var body struct {
				Status api.VerificationStatus `json:"status"`
			}
			_ = json.NewDecoder(req.Body).Decode(&body)

			backend.mu.Lock()
			backend.reports[verificationID] = body.Status
			backend.mu.Unlock()

			w.WriteHeader(http.StatusNoContent)
		},
	)

	mux.HandleFunc(
		"POST /api/v1/agent/verification/{agentId}/heartbeat",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"lastSeenAt":           time.Now().UTC(),
				"abortVerificationIds": []uuid.UUID{},
			})
		},
	)

	return backend, httptest.NewServer(mux)
}

func (b *concurrentRestoreBackend) reportsSnapshot() map[uuid.UUID]api.VerificationStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make(map[uuid.UUID]api.VerificationStatus, len(b.reports))
	for id, status := range b.reports {
		out[id] = status
	}

	return out
}

type recordingContainer struct {
	id              int
	inContainerConn dbconn.Conn
	verifierConn    dbconn.Conn
	terminated      atomic.Bool
}

func (c *recordingContainer) Exec(
	_ context.Context, _ []string, stdin io.Reader, _ []string,
) (restore.ExecResult, error) {
	if stdin != nil {
		_, _ = io.Copy(io.Discard, stdin)
	}

	return restore.ExecResult{}, nil
}

func (c *recordingContainer) GetInContainerConn() dbconn.Conn { return c.inContainerConn }
func (c *recordingContainer) GetVerifierConn() dbconn.Conn    { return c.verifierConn }

func (c *recordingContainer) GetDiskUsageBytes(context.Context) (int64, error) {
	return 0, nil
}

func (c *recordingContainer) Terminate(context.Context) error {
	c.terminated.Store(true)
	return nil
}

type recordingSpawner struct {
	nextID atomic.Int32

	mu         sync.Mutex
	containers []*recordingContainer
}

func (s *recordingSpawner) Spawn(_ context.Context, _ SpawnRequest) (JobContainer, error) {
	c := &recordingContainer{
		id:              int(s.nextID.Add(1)),
		inContainerConn: testConn(),
		verifierConn:    testConn(),
	}

	s.mu.Lock()
	s.containers = append(s.containers, c)
	s.mu.Unlock()

	return c, nil
}

func (s *recordingSpawner) snapshot() []*recordingContainer {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*recordingContainer, len(s.containers))
	copy(out, s.containers)

	return out
}

type barrierRestorer struct {
	barrier *sync.WaitGroup
}

func (r *barrierRestorer) StageBackupViaExec(
	_ context.Context, _ restore.ExecRunner, body io.Reader, _ string,
) error {
	if body != nil {
		_, _ = io.Copy(io.Discard, body)
	}

	return nil
}

func (r *barrierRestorer) RunPgRestore(
	_ context.Context, _ restore.ExecRunner, _ string, _ dbconn.Conn, _ int,
) (restore.Result, error) {
	r.barrier.Done()
	r.barrier.Wait()

	return restore.Result{PgRestoreExitCode: 0, DurationMs: 1}, nil
}

func (r *barrierRestorer) RunTimescalePreRestore(context.Context, dbconn.Conn) error  { return nil }
func (r *barrierRestorer) RunTimescalePostRestore(context.Context, dbconn.Conn) error { return nil }

func makeConcurrentAssignment(id uuid.UUID) *api.JobAssignment {
	return &api.JobAssignment{
		VerificationID:     id,
		BackupID:           uuid.New(),
		BackupSizeMb:       1,
		MaxContainerDiskMb: 0,
		Database: api.AssignedDatabase{
			Type:              "POSTGRES_LOGICAL",
			PostgresqlLogical: &api.AssignedPostgresql{Version: "16"},
		},
	}
}

func Test_Run_WhenTwoVerificationsClaimed_EachGetsIsolatedContainerAndBothComplete(
	t *testing.T,
) {
	id1, id2 := uuid.New(), uuid.New()
	claims := []*api.JobAssignment{
		makeConcurrentAssignment(id1),
		makeConcurrentAssignment(id2),
	}

	backend, server := newConcurrentRestoreBackend(claims)
	t.Cleanup(server.Close)

	client := api.NewClient(server.URL, "", uuid.NewString(), testutil.DiscardLogger())
	heartbeater := heartbeat.NewHeartbeater(client, testCapacity(), testutil.DiscardLogger())

	spawner := &recordingSpawner{}
	barrier := &sync.WaitGroup{}
	barrier.Add(2)

	runnerUnderTest := NewRunner(
		client, testCapacity(), NewPool(2),
		spawner,
		&barrierRestorer{barrier: barrier},
		&fakeStats{stats: verifier.Stats{DBSizeBytes: 1, SchemaCount: 1, TableCount: 1}},
		heartbeater,
		testutil.DiscardLogger(),
	)
	runnerUnderTest.connAlive = func(context.Context, dbconn.Conn) bool { return true }

	runCtx, runCancel := context.WithCancel(t.Context())
	runDone := make(chan struct{})
	go func() {
		runnerUnderTest.Run(runCtx)
		close(runDone)
	}()

	require.Eventually(t, func() bool {
		snapshot := backend.reportsSnapshot()
		return snapshot[id1] == api.VerificationStatusCompleted &&
			snapshot[id2] == api.VerificationStatusCompleted
	}, 10*time.Second, 20*time.Millisecond,
		"both verifications must complete via the real claim loop + NewPool(2) fan-out; "+
			"the barrier in pg_restore would deadlock unless both jobs were truly in flight at once")

	runCancel()
	select {
	case <-runDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	spawned := spawner.snapshot()
	require.Len(t, spawned, 2,
		"each claimed verification must spawn its own container — proves isolation, not reuse")
	assert.NotEqual(t, spawned[0].id, spawned[1].id,
		"the two spawned containers must be distinct instances")
	for _, c := range spawned {
		assert.True(t, c.terminated.Load(),
			"every spawned container must be torn down (deferred Terminate in executeJob)")
	}

	reports := backend.reportsSnapshot()
	assert.Equal(t, api.VerificationStatusCompleted, reports[id1])
	assert.Equal(t, api.VerificationStatusCompleted, reports[id2])
}

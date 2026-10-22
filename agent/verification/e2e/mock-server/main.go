// Mock Databasus API server for the verification agent e2e: serves version
// checks, the verification-agent binary download, the capacity heartbeat, and
// the job protocol (claim / backup-stream / report) with fault-injection
// controls. Stdlib only — built standalone via `go build main.go`.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type server struct {
	mu         sync.RWMutex
	version    string
	binaryPath string
	heartbeats int
	lastBody   map[string]any

	// job protocol state
	claimJob          map[string]any
	backupFixturePath string
	reports           []map[string]any
	abortIDs          []string
	streamFailRemain  int
	tearStreamOnce    bool
	reportGone        bool
	streamDelay       time.Duration
}

func main() {
	s := &server{version: "v1.0.0", binaryPath: "/artifacts/agent-v1"}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/system/version", s.handleVersion)
	mux.HandleFunc("GET /api/v1/system/verification-agent", s.handleDownload)
	mux.HandleFunc("POST /api/v1/agent/verification/{agentId}/heartbeat", s.handleHeartbeat)

	mux.HandleFunc("POST /api/v1/agent/verifications/{agentId}/claim", s.handleClaim)
	mux.HandleFunc("GET /api/v1/agent/verifications/{agentId}/{id}/backup-stream", s.handleBackupStream)
	mux.HandleFunc("POST /api/v1/agent/verifications/{agentId}/{id}/report", s.handleReport)

	mux.HandleFunc("POST /mock/set-version", s.handleSetVersion)
	mux.HandleFunc("POST /mock/set-binary-path", s.handleSetBinaryPath)
	mux.HandleFunc("GET /mock/heartbeats", s.handleHeartbeatStatus)
	mux.HandleFunc("POST /mock/set-claim", s.handleSetClaim)
	mux.HandleFunc("POST /mock/set-backup-fixture", s.handleSetBackupFixture)
	mux.HandleFunc("POST /mock/set-abort", s.handleSetAbort)
	mux.HandleFunc("POST /mock/set-stream-fail", s.handleSetStreamFail)
	mux.HandleFunc("POST /mock/set-stream-delay", s.handleSetStreamDelay)
	mux.HandleFunc("POST /mock/set-report-gone", s.handleSetReportGone)
	mux.HandleFunc("GET /mock/reports", s.handleGetReports)
	mux.HandleFunc("POST /mock/reset", s.handleReset)
	mux.HandleFunc("GET /health", s.handleHealth)

	addr := ":4050"
	log.Printf("mock server starting on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func (s *server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	version := s.version
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"version": version})
}

func (s *server) handleDownload(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	path := s.binaryPath
	version := s.version
	s.mu.RUnlock()

	log.Printf("GET verification-agent (arch=%s) -> %s", r.URL.Query().Get("arch"), path)
	w.Header().Set("X-Databasus-Version", version)
	http.ServeFile(w, r, path)
}

func (s *server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.heartbeats++
	s.lastBody = body
	count := s.heartbeats
	abortIDs := append([]string{}, s.abortIDs...)
	s.mu.Unlock()

	log.Printf("POST heartbeat (agent=%s) -> count=%d", r.PathValue("agentId"), count)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"lastSeenAt":           time.Now().UTC().Format(time.RFC3339),
		"abortVerificationIds": abortIDs,
	})
}

func (s *server) handleClaim(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	job := s.claimJob
	s.claimJob = nil // one-shot: a single verification per test run
	s.mu.Unlock()

	if job == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	log.Printf("POST claim (agent=%s) -> assigning %v", r.PathValue("agentId"), job["verificationId"])
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (s *server) handleBackupStream(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	failRemain := s.streamFailRemain
	if failRemain > 0 {
		s.streamFailRemain--
	}
	tear := s.tearStreamOnce
	s.tearStreamOnce = false
	fixture := s.backupFixturePath
	delay := s.streamDelay
	s.mu.Unlock()

	// The delay holds the response headers back, so the agent's restore
	// container stays up waiting for its backup while a test inspects it.
	if delay > 0 {
		log.Printf("GET backup-stream -> delaying %s", delay)

		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
	}

	if failRemain > 0 {
		log.Printf("GET backup-stream -> injected 503 (remaining=%d)", failRemain-1)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if tear {
		log.Printf("GET backup-stream -> tearing connection mid-body")
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("PARTIAL"))

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		if hj, ok := w.(http.Hijacker); ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}

		return
	}

	log.Printf("GET backup-stream (id=%s) -> %s", r.PathValue("id"), fixture)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, fixture)
}

func (s *server) handleReport(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body["verificationId"] = r.PathValue("id")

	s.mu.Lock()
	gone := s.reportGone
	if !gone {
		s.reports = append(s.reports, body)
	}
	s.mu.Unlock()

	if gone {
		log.Printf("POST report (id=%s) -> 410 gone", r.PathValue("id"))
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"reason":"gone"}`))
		return
	}

	log.Printf("POST report (id=%s) -> status=%v exit=%v", r.PathValue("id"), body["status"], body["pgRestoreExitCode"])
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleSetVersion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.version = body.Version
	s.mu.Unlock()

	log.Printf("POST /mock/set-version -> %s", body.Version)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetBinaryPath(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BinaryPath string `json:"binaryPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.binaryPath = body.BinaryPath
	s.mu.Unlock()

	log.Printf("POST /mock/set-binary-path -> %s", body.BinaryPath)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetClaim(w http.ResponseWriter, r *http.Request) {
	var job map[string]any
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.claimJob = job
	s.mu.Unlock()

	log.Printf("POST /mock/set-claim -> %v", job["verificationId"])
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetBackupFixture(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(body.Path); err != nil {
		http.Error(w, "fixture not found: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.backupFixturePath = body.Path
	s.mu.Unlock()

	log.Printf("POST /mock/set-backup-fixture -> %s", body.Path)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetAbort(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.abortIDs = body.IDs
	s.mu.Unlock()

	log.Printf("POST /mock/set-abort -> %v", body.IDs)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetStreamFail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Count          int  `json:"count"`
		TearStreamOnce bool `json:"tearStreamOnce"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.streamFailRemain = body.Count
	s.tearStreamOnce = body.TearStreamOnce
	s.mu.Unlock()

	log.Printf("POST /mock/set-stream-fail -> count=%d tear=%v", body.Count, body.TearStreamOnce)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetStreamDelay(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Seconds int `json:"seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.streamDelay = time.Duration(body.Seconds) * time.Second
	s.mu.Unlock()

	log.Printf("POST /mock/set-stream-delay -> %ds", body.Seconds)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleSetReportGone(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Gone bool `json:"gone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.reportGone = body.Gone
	s.mu.Unlock()

	log.Printf("POST /mock/set-report-gone -> %v", body.Gone)
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleGetReports(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count":   len(s.reports),
		"reports": s.reports,
	})
}

func (s *server) handleHeartbeatStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count": s.heartbeats,
		"last":  s.lastBody,
	})
}

// handleReset wipes per-test state (reports, claim, abort/stream/report-gone
// flags, stream delay, fixture path). Version + binary path are left alone — pin them via
// /mock/set-version + /mock/set-binary-path. Each restore-style e2e test
// should call this at the top to avoid seeing prior-test artifacts.
func (s *server) handleReset(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.reports = nil
	s.claimJob = nil
	s.backupFixturePath = ""
	s.abortIDs = nil
	s.streamFailRemain = 0
	s.tearStreamOnce = false
	s.reportGone = false
	s.streamDelay = 0
	s.heartbeats = 0
	s.lastBody = nil
	s.mu.Unlock()

	log.Printf("POST /mock/reset -> state wiped")
	w.WriteHeader(http.StatusOK)
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

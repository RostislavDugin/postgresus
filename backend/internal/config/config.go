package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	env_utils "databasus-backend/internal/util/env"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/tools"
)

var log = logger.GetLogger()

const (
	AppModeWeb        = "web"
	AppModeBackground = "background"
)

const (
	// defaultTestParallelWorkers is the number of parallel test workers, used
	// when TEST_PARALLEL_WORKERS is unset. Each worker gets its own metadata DB
	// and Valkey logical DB, so this must stay <= 16 (Valkey's default logical DB
	// count) and equal to the `go test -p` value.
	defaultTestParallelWorkers = 8

	// testSlotAdvisoryLockBase is the first pg_advisory_lock key reserved for
	// test-worker slot claims; slot N uses base+N.
	testSlotAdvisoryLockBase = 945_000_000

	// testSlotClaimTimeout bounds the wait for a free slot, absorbing the brief
	// window where `go test -p` starts the next package before the previous
	// process has exited and released its advisory lock.
	testSlotClaimTimeout = 60 * time.Second
)

type EnvVariables struct {
	IsTesting bool
	EnvMode   env_utils.EnvMode `env:"ENV_MODE" required:"true"`

	// Internal database
	DatabaseDsn         string `env:"DATABASE_DSN"          required:"true"`
	TestDatabaseDsn     string `env:"TEST_DATABASE_DSN"`
	TestParallelWorkers int    `env:"TEST_PARALLEL_WORKERS"`
	// Internal Valkey
	ValkeyHost     string `env:"VALKEY_HOST"     required:"true"`
	ValkeyPort     string `env:"VALKEY_PORT"     required:"true"`
	ValkeyUsername string `env:"VALKEY_USERNAME"`
	ValkeyPassword string `env:"VALKEY_PASSWORD"`
	ValkeyIsSsl    bool   `env:"VALKEY_IS_SSL"   required:"true"`

	// Per-worker test isolation (computed, only set under `go test`): each test
	// binary claims a slot 0..TestParallelWorkers-1 that selects its own metadata DB
	// (DatabaseDsn is rewritten to dbname=<base>_w{slot}), its own Valkey logical
	// DB (ValkeySelectDB), and a cache-key namespace ("w{slot}:"). All zero/empty in
	// production.
	ValkeySelectDB int
	CacheNamespace string

	TestLocalhost string `env:"TEST_LOCALHOST"`

	ShowDbInstallationVerificationLogs bool `env:"SHOW_DB_INSTALLATION_VERIFICATION_LOGS"`

	DataFolder            string
	TempFolder            string
	SecretKeyPath         string
	TelemetryInstancePath string

	IsDisableAnonymousTelemetry bool `env:"IS_DISABLE_ANONYMOUS_TELEMETRY"`

	// TestLogicalPostgres16Port is the shared always-on logical-Postgres fixture used by
	// GetTestPostgresConfig / CreateTestDatabase across the test suite. The per-version PG
	// logical tests (14-18 + SSL + mTLS) run on testcontainers and need no fixed port.
	TestLogicalPostgres16Port string `env:"TEST_LOGICAL_POSTGRES_16_PORT"`

	// The physical primary sources are the shared always-on fixture (like the logical PG-16
	// fixture); the per-version, no-summary, tablespace, mTLS and restore-target containers all run
	// on testcontainers and need no fixed port.
	TestPhysicalPostgres17Port string `env:"TEST_PHYSICAL_POSTGRES_17_PORT"`
	TestPhysicalPostgres18Port string `env:"TEST_PHYSICAL_POSTGRES_18_PORT"`

	// oauth
	GitHubClientID     string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`

	// Cloudflare Turnstile
	CloudflareTurnstileSecretKey string `env:"CLOUDFLARE_TURNSTILE_SECRET_KEY"`
	CloudflareTurnstileSiteKey   string `env:"CLOUDFLARE_TURNSTILE_SITE_KEY"`

	// SMTP configuration (optional)
	SMTPHost               string `env:"SMTP_HOST"`
	SMTPPort               int    `env:"SMTP_PORT"`
	SMTPUser               string `env:"SMTP_USER"`
	SMTPPassword           string `env:"SMTP_PASSWORD"`
	SMTPFrom               string `env:"SMTP_FROM"`
	SMTPInsecureSkipVerify bool   `env:"SMTP_INSECURE_SKIP_VERIFY"`

	// Application URL (optional) - used for email links
	DatabasusURL string `env:"DATABASUS_URL"`
}

var env EnvVariables

var initEnv = sync.OnceFunc(loadEnvVariables)

func GetEnv() *EnvVariables {
	initEnv()
	return &env
}

func loadEnvVariables() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Warn("could not get current working directory", "error", err)
		cwd = "."
	}

	backendRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(backendRoot, "go.mod")); err == nil {
			break
		}

		parent := filepath.Dir(backendRoot)
		if parent == backendRoot {
			break
		}

		backendRoot = parent
	}

	envPath := filepath.Join(filepath.Dir(backendRoot), ".env")

	log.Info("trying to load .env", "path", envPath)
	if err := godotenv.Load(envPath); err != nil {
		log.Error("error loading .env file from repo root", "path", envPath, "error", err)
		logger.ExitAfterFlush(1)
	}
	log.Info("successfully loaded .env", "path", envPath)

	// Empty values for non-string fields (e.g. SMTP_PORT=) crash cleanenv's
	// strconv parsing. Drop them so cleanenv falls back to the Go zero value.
	unsetEmptyEnvVars()

	err = cleanenv.ReadEnv(&env)
	if err != nil {
		log.Error("configuration could not be loaded", "error", err)
		logger.ExitAfterFlush(1)
	}

	if env.SMTPHost != "" && env.SMTPPort <= 0 {
		log.Error("SMTP_PORT must be a positive integer when SMTP_HOST is set", "value", env.SMTPPort)
		logger.ExitAfterFlush(1)
	}

	// Set default value for ShowDbInstallationVerificationLogs if not defined
	if os.Getenv("SHOW_DB_INSTALLATION_VERIFICATION_LOGS") == "" {
		env.ShowDbInstallationVerificationLogs = true
	}

	for _, arg := range os.Args {
		if strings.Contains(arg, "test") {
			env.IsTesting = true
			break
		}
	}

	if env.IsTesting {
		if env.TestDatabaseDsn == "" {
			log.Error("TEST_DATABASE_DSN is empty")
			logger.ExitAfterFlush(1)
		}

		env.DatabaseDsn = env.TestDatabaseDsn

		if env.TestParallelWorkers <= 0 {
			env.TestParallelWorkers = defaultTestParallelWorkers
		}

		// Only a real `go test` binary claims a per-worker slot; the cleanup_test_db
		// command and any other tool run with IsTesting=true must operate on all
		// slots, so they keep the base test DSN and default Valkey DB.
		if strings.Contains(os.Args[0], ".test") {
			applyTestWorkerSlot()
		}
	}

	if env.DatabaseDsn == "" {
		log.Error("DATABASE_DSN is empty")
		logger.ExitAfterFlush(1)
	}

	if env.EnvMode == "" {
		log.Error("ENV_MODE is empty")
		logger.ExitAfterFlush(1)
	}
	if env.EnvMode != "development" && env.EnvMode != "production" {
		log.Error("ENV_MODE is invalid", "mode", env.EnvMode)
		logger.ExitAfterFlush(1)
	}
	log.Info("ENV_MODE loaded", "mode", env.EnvMode)

	tools.LogAndExitIfClientToolsBroken(log, env.ShowDbInstallationVerificationLogs)

	if env.TestLocalhost == "" {
		env.TestLocalhost = "localhost"
	}

	// Valkey
	if env.ValkeyHost == "" {
		log.Error("VALKEY_HOST is empty")
		logger.ExitAfterFlush(1)
	}
	if env.ValkeyPort == "" {
		log.Error("VALKEY_PORT is empty")
		logger.ExitAfterFlush(1)
	}

	// Store the data and temp folders one level below the root
	// (projectRoot/databasus-data -> /databasus-data)
	env.DataFolder = filepath.Join(filepath.Dir(backendRoot), "databasus-data", "backups")
	env.TempFolder = filepath.Join(filepath.Dir(backendRoot), "databasus-data", "temp")
	env.SecretKeyPath = filepath.Join(filepath.Dir(backendRoot), "databasus-data", "secret.key")
	env.TelemetryInstancePath = filepath.Join(
		filepath.Dir(backendRoot), "databasus-data", "instance.json",
	)

	if env.IsTesting {
		if env.TestLogicalPostgres16Port == "" {
			log.Error("TEST_LOGICAL_POSTGRES_16_PORT is empty")
			logger.ExitAfterFlush(1)
		}
		if env.TestPhysicalPostgres17Port == "" {
			log.Error("TEST_PHYSICAL_POSTGRES_17_PORT is empty")
			logger.ExitAfterFlush(1)
		}
		if env.TestPhysicalPostgres18Port == "" {
			log.Error("TEST_PHYSICAL_POSTGRES_18_PORT is empty")
			logger.ExitAfterFlush(1)
		}
	}

	log.Info("environment variables loaded successfully")
}

func unsetEmptyEnvVars() {
	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}

		if value == "" {
			_ = os.Unsetenv(key)
		}
	}
}

// slotLockConn holds the system-DB connection whose session owns this worker's
// advisory lock. It must live for the whole process: closing it (or letting it
// be garbage-collected) releases the lock and frees the slot for another worker
// mid-run. It is assigned-only by design — the reference itself is the point, so
// it anchors the connection lifetime as a GC root.
//
//nolint:unused // assigned-only: keeps the advisory-lock connection alive for the process lifetime
var slotLockConn *sql.Conn

// applyTestWorkerSlot claims a free slot for this test binary and rewrites the
// env so the worker runs fully isolated: its own metadata DB, Valkey logical DB,
// and registry namespace.
func applyTestWorkerSlot() {
	baseDbName, _, err := RewriteDbName(env.TestDatabaseDsn, systemDbName)
	if err != nil {
		log.Error("could not parse TEST_DATABASE_DSN for slot isolation", "error", err)
		logger.ExitAfterFlush(1)
	}

	slot := claimTestWorkerSlot(env.TestDatabaseDsn, env.TestParallelWorkers)

	slotDbName := fmt.Sprintf("%s_w%d", baseDbName, slot)
	_, slotDsn, err := RewriteDbName(env.TestDatabaseDsn, slotDbName)
	if err != nil {
		log.Error("could not build per-slot DSN", "error", err)
		logger.ExitAfterFlush(1)
	}

	env.DatabaseDsn = slotDsn
	env.ValkeySelectDB = slot
	env.CacheNamespace = fmt.Sprintf("w%d:", slot)

	log.Info("claimed test worker slot", "slot", slot, "db", slotDbName)
}

// systemDbName is the always-present database used for slot coordination and as
// the connection target when (re)creating per-slot databases.
const systemDbName = "postgres"

// claimTestWorkerSlot acquires a session-level advisory lock for the first free
// slot in [0,pool) on the system Postgres DB and returns it. The lock is held by
// slotLockConn until the process exits. It retries until testSlotClaimTimeout to
// absorb the handoff window where `go test -p` overlaps a finishing worker.
func claimTestWorkerSlot(testDsn string, pool int) int {
	_, systemDsn, err := RewriteDbName(testDsn, systemDbName)
	if err != nil {
		log.Error("could not build system DSN for slot claim", "error", err)
		logger.ExitAfterFlush(1)
	}

	db, err := sql.Open("pgx", systemDsn)
	if err != nil {
		log.Error("could not open system DB for slot claim", "error", err)
		logger.ExitAfterFlush(1)
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		log.Error("could not get system DB connection for slot claim", "error", err)
		logger.ExitAfterFlush(1)
	}

	deadline := time.Now().Add(testSlotClaimTimeout)
	for {
		for slot := range pool {
			var locked bool
			lockErr := conn.QueryRowContext(
				ctx,
				"SELECT pg_try_advisory_lock($1)",
				testSlotAdvisoryLockBase+int64(slot),
			).Scan(&locked)
			if lockErr != nil {
				log.Error("advisory lock query failed during slot claim", "error", lockErr)
				logger.ExitAfterFlush(1)
			}

			if locked {
				slotLockConn = conn
				return slot
			}
		}

		if time.Now().After(deadline) {
			log.Error(
				"no free test DB slot",
				"pool", pool,
				"hint", "TEST_PARALLEL_WORKERS must be >= the `go test -p` value",
			)
			logger.ExitAfterFlush(1)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// RewriteDbName replaces the dbname= token in a keyword/value Postgres DSN,
// returning the original dbname and the rewritten DSN. Used both to derive the
// system DSN and to build per-slot test DSNs.
func RewriteDbName(dsn, newDbName string) (origDbName, rewritten string, err error) {
	parts := strings.Fields(dsn)
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return "", "", fmt.Errorf("invalid DSN token: %q", p)
		}

		if k == "dbname" {
			origDbName = v
			out = append(out, "dbname="+newDbName)
			continue
		}

		out = append(out, p)
	}

	if origDbName == "" {
		return "", "", fmt.Errorf("DSN missing dbname: %q", dsn)
	}

	return origDbName, strings.Join(out, " "), nil
}

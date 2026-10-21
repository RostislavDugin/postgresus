package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	devBackendURL = "http://localhost:4005"

	// The command runs in backend/, next to the repository's .env.
	envFilePath = "../.env"

	// Creating a WAL-streaming database runs the replication readiness checks on the source,
	// which takes longer than an ordinary request.
	requestTimeout = 2 * time.Minute
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	params, err := loadSeedParams()
	if err != nil {
		log.Error("cannot read the seed settings", "error", err)
		os.Exit(1)
	}

	os.Exit(runSeed(context.Background(), &http.Client{Timeout: requestTimeout}, params, log))
}

func runSeed(ctx context.Context, httpClient *http.Client, params seedParams, log *slog.Logger) int {
	if err := seedDevApp(ctx, httpClient, params, log); err != nil {
		log.Error("seed failed", "error", err)

		return 1
	}

	return 0
}

// The defaults are the ones docker-compose.yml falls back to, so a .env without a variable still
// points at the container compose started.
func loadSeedParams() (seedParams, error) {
	if err := godotenv.Load(envFilePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return seedParams{}, fmt.Errorf("load %s: %w", envFilePath, err)
	}

	if envMode := os.Getenv("ENV_MODE"); envMode != "development" {
		return seedParams{}, fmt.Errorf("ENV_MODE is %q, and the seed runs only in development", envMode)
	}

	logicalPort, err := strconv.Atoi(readEnvOrDefault("TEST_LOGICAL_POSTGRES_16_PORT", "5004"))
	if err != nil {
		return seedParams{}, fmt.Errorf("TEST_LOGICAL_POSTGRES_16_PORT: %w", err)
	}

	walPort, err := strconv.Atoi(readEnvOrDefault("WAL_BACKUP_POSTGRES_18_PORT", "5018"))
	if err != nil {
		return seedParams{}, fmt.Errorf("WAL_BACKUP_POSTGRES_18_PORT: %w", err)
	}

	sourceHost := readEnvOrDefault("TEST_LOCALHOST", "localhost")

	return seedParams{
		BaseURL:             devBackendURL,
		Email:               "test@email.com",
		Password:            "Q1234567",
		WorkspaceName:       "Dev",
		StorageName:         "Local",
		LogicalDatabaseName: "Logical PG16",
		LogicalSource: postgresSource{
			Host:     sourceHost,
			Port:     logicalPort,
			Username: "testuser",
			Password: "testpassword",
			Database: "testdb",
		},
		WalDatabaseName: "WAL PG18",
		WalSource: postgresSource{
			Host:     sourceHost,
			Port:     walPort,
			Username: readEnvOrDefault("WAL_BACKUP_DB_USER", "testuser"),
			Password: readEnvOrDefault("WAL_BACKUP_DB_PASSWORD", "testpassword"),
		},
	}, nil
}

func readEnvOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return defaultValue
}

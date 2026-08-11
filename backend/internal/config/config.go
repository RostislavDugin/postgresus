package config

import (
	"os"
	"path/filepath"
	env_utils "postgresus-backend/internal/util/env"
	"postgresus-backend/internal/util/logger"
	"postgresus-backend/internal/util/tools"
	"strings"
	"sync"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

var log = logger.GetLogger()

const (
	AppModeWeb        = "web"
	AppModeBackground = "background"
)

type EnvVariables struct {
	IsTesting            bool
	DatabaseDsn          string            `env:"DATABASE_DSN"         required:"true"`
	EnvMode              env_utils.EnvMode `env:"ENV_MODE"             required:"true"`
	PostgresesInstallDir string            `env:"POSTGRES_INSTALL_DIR"`

	DataFolder    string
	TempFolder    string
	SecretKeyPath string

	TestPostgres12Addr string `env:"TEST_POSTGRES_12_ADDR"`
	TestPostgres13Addr string `env:"TEST_POSTGRES_13_ADDR"`
	TestPostgres14Addr string `env:"TEST_POSTGRES_14_ADDR"`
	TestPostgres15Addr string `env:"TEST_POSTGRES_15_ADDR"`
	TestPostgres16Addr string `env:"TEST_POSTGRES_16_ADDR"`
	TestPostgres17Addr string `env:"TEST_POSTGRES_17_ADDR"`
	TestPostgres18Addr string `env:"TEST_POSTGRES_18_ADDR"`

	TestMinioAddr   string `env:"TEST_MINIO_ADDR"`
	TestAzuriteAddr string `env:"TEST_AZURITE_ADDR"`
	TestNasAddr     string `env:"TEST_NAS_ADDR"`
	TestFtpAddr     string `env:"TEST_FTP_ADDR"`

	// oauth
	GitHubClientID     string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
}

var (
	env  EnvVariables
	once sync.Once
)

func GetEnv() EnvVariables {
	once.Do(loadEnvVariables)
	return env
}

func loadEnvVariables() {
	// Get current working directory
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

	envPaths := []string{
		filepath.Join(cwd, ".env"),
		filepath.Join(backendRoot, ".env"),
	}

	var loaded bool
	for _, path := range envPaths {
		log.Info("Trying to load .env", "path", path)
		if err := godotenv.Load(path); err == nil {
			log.Info("Successfully loaded .env", "path", path)
			loaded = true
			break
		}
	}

	if !loaded {
		log.Error("Error loading .env file: could not find .env in any location")
		os.Exit(1)
	}

	err = cleanenv.ReadEnv(&env)
	if err != nil {
		log.Error("Configuration could not be loaded", "error", err)
		os.Exit(1)
	}

	for _, arg := range os.Args {
		if strings.Contains(arg, "test") {
			env.IsTesting = true
			break
		}
	}

	if env.DatabaseDsn == "" {
		log.Error("DATABASE_DSN is empty")
		os.Exit(1)
	}

	if env.EnvMode == "" {
		log.Error("ENV_MODE is empty")
		os.Exit(1)
	}
	if env.EnvMode != "development" && env.EnvMode != "production" {
		log.Error("ENV_MODE is invalid", "mode", env.EnvMode)
		os.Exit(1)
	}
	log.Info("ENV_MODE loaded", "mode", env.EnvMode)

	env.PostgresesInstallDir = filepath.Join(backendRoot, "tools", "postgresql")
	tools.VerifyPostgresesInstallation(log, env.EnvMode, env.PostgresesInstallDir)

	// Store the data and temp folders one level below the root
	// (projectRoot/postgresus-data -> /postgresus-data)
	env.DataFolder = filepath.Join(filepath.Dir(backendRoot), "postgresus-data", "backups")
	env.TempFolder = filepath.Join(filepath.Dir(backendRoot), "postgresus-data", "temp")
	env.SecretKeyPath = filepath.Join(filepath.Dir(backendRoot), "postgresus-data", "secret.key")

	if env.IsTesting {
		required := map[string]string{
			"TEST_POSTGRES_12_ADDR": env.TestPostgres12Addr,
			"TEST_POSTGRES_13_ADDR": env.TestPostgres13Addr,
			"TEST_POSTGRES_14_ADDR": env.TestPostgres14Addr,
			"TEST_POSTGRES_15_ADDR": env.TestPostgres15Addr,
			"TEST_POSTGRES_16_ADDR": env.TestPostgres16Addr,
			"TEST_POSTGRES_17_ADDR": env.TestPostgres17Addr,
			"TEST_POSTGRES_18_ADDR": env.TestPostgres18Addr,
			"TEST_MINIO_ADDR":       env.TestMinioAddr,
			"TEST_AZURITE_ADDR":     env.TestAzuriteAddr,
			"TEST_NAS_ADDR":         env.TestNasAddr,
			"TEST_FTP_ADDR":         env.TestFtpAddr,
		}

		for name, value := range required {
			if value == "" {
				log.Error("required test environment variable is empty", "variable", name)
				os.Exit(1)
			}
		}
	}

	log.Info("Environment variables loaded successfully!")
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
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

type EnvVariables struct {
	IsTesting bool
	EnvMode   env_utils.EnvMode `env:"ENV_MODE" required:"true"`

	// Internal database
	DatabaseDsn string `env:"DATABASE_DSN" required:"true"`
	// Internal Valkey
	ValkeyHost     string `env:"VALKEY_HOST"     required:"true"`
	ValkeyPort     string `env:"VALKEY_PORT"     required:"true"`
	ValkeyUsername string `env:"VALKEY_USERNAME"`
	ValkeyPassword string `env:"VALKEY_PASSWORD"`
	ValkeyIsSsl    bool   `env:"VALKEY_IS_SSL"   required:"true"`

	IsCloud       bool   `env:"IS_CLOUD"`
	TestLocalhost string `env:"TEST_LOCALHOST"`

	ShowDbInstallationVerificationLogs bool `env:"SHOW_DB_INSTALLATION_VERIFICATION_LOGS"`

	IsManyNodesMode          bool `env:"IS_MANY_NODES_MODE"`
	IsPrimaryNode            bool `env:"IS_PRIMARY_NODE"`
	IsProcessingNode         bool `env:"IS_PROCESSING_NODE"`
	NodeNetworkThroughputMBs int  `env:"NODE_NETWORK_THROUGHPUT_MBPS"`

	DataFolder            string
	TempFolder            string
	SecretKeyPath         string
	TelemetryInstancePath string

	IsDisableAnonymousTelemetry bool `env:"IS_DISABLE_ANONYMOUS_TELEMETRY"`

	// Billing (tax-exclusive)
	PricePerGBCents int64 `env:"PRICE_PER_GB_CENTS"`
	MinStorageGB    int
	MaxStorageGB    int
	TrialDuration   time.Duration
	TrialStorageGB  int
	GracePeriod     time.Duration
	// Paddle billing
	IsPaddleSandbox     bool   `env:"IS_PADDLE_SANDBOX"`
	PaddleApiKey        string `env:"PADDLE_API_KEY"`
	PaddleWebhookSecret string `env:"PADDLE_WEBHOOK_SECRET"`
	PaddlePriceID       string `env:"PADDLE_PRICE_ID"`
	PaddleClientToken   string `env:"PADDLE_CLIENT_TOKEN"`

	TestPostgres12Port string `env:"TEST_POSTGRES_12_PORT"`
	TestPostgres13Port string `env:"TEST_POSTGRES_13_PORT"`
	TestPostgres14Port string `env:"TEST_POSTGRES_14_PORT"`
	TestPostgres15Port string `env:"TEST_POSTGRES_15_PORT"`
	TestPostgres16Port string `env:"TEST_POSTGRES_16_PORT"`
	TestPostgres17Port string `env:"TEST_POSTGRES_17_PORT"`
	TestPostgres18Port string `env:"TEST_POSTGRES_18_PORT"`

	TestMinioPort        string `env:"TEST_MINIO_PORT"`
	TestMinioConsolePort string `env:"TEST_MINIO_CONSOLE_PORT"`

	TestAzuriteBlobPort string `env:"TEST_AZURITE_BLOB_PORT"`

	TestNASPort  string `env:"TEST_NAS_PORT"`
	TestFTPPort  string `env:"TEST_FTP_PORT"`
	TestSFTPPort string `env:"TEST_SFTP_PORT"`

	TestMysql57Port string `env:"TEST_MYSQL_57_PORT"`
	TestMysql80Port string `env:"TEST_MYSQL_80_PORT"`
	TestMysql84Port string `env:"TEST_MYSQL_84_PORT"`
	TestMysql90Port string `env:"TEST_MYSQL_90_PORT"`

	TestMariadb55Port   string `env:"TEST_MARIADB_55_PORT"`
	TestMariadb101Port  string `env:"TEST_MARIADB_101_PORT"`
	TestMariadb102Port  string `env:"TEST_MARIADB_102_PORT"`
	TestMariadb103Port  string `env:"TEST_MARIADB_103_PORT"`
	TestMariadb104Port  string `env:"TEST_MARIADB_104_PORT"`
	TestMariadb105Port  string `env:"TEST_MARIADB_105_PORT"`
	TestMariadb106Port  string `env:"TEST_MARIADB_106_PORT"`
	TestMariadb1011Port string `env:"TEST_MARIADB_1011_PORT"`
	TestMariadb114Port  string `env:"TEST_MARIADB_114_PORT"`
	TestMariadb118Port  string `env:"TEST_MARIADB_118_PORT"`
	TestMariadb120Port  string `env:"TEST_MARIADB_120_PORT"`

	TestMongodb42Port string `env:"TEST_MONGODB_42_PORT"`
	TestMongodb44Port string `env:"TEST_MONGODB_44_PORT"`
	TestMongodb50Port string `env:"TEST_MONGODB_50_PORT"`
	TestMongodb60Port string `env:"TEST_MONGODB_60_PORT"`
	TestMongodb70Port string `env:"TEST_MONGODB_70_PORT"`
	TestMongodb82Port string `env:"TEST_MONGODB_82_PORT"`

	// oauth
	GitHubClientID     string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`

	// Cloudflare Turnstile
	CloudflareTurnstileSecretKey string `env:"CLOUDFLARE_TURNSTILE_SECRET_KEY"`
	CloudflareTurnstileSiteKey   string `env:"CLOUDFLARE_TURNSTILE_SITE_KEY"`

	// SMTP configuration (optional)
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT"`
	SMTPUser     string `env:"SMTP_USER"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM"`

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

	log.Info("Trying to load .env", "path", envPath)
	if err := godotenv.Load(envPath); err != nil {
		log.Error("Error loading .env file from repo root", "path", envPath, "error", err)
		os.Exit(1)
	}
	log.Info("Successfully loaded .env", "path", envPath)

	// Empty values for non-string fields (e.g. SMTP_PORT=) crash cleanenv's
	// strconv parsing. Drop them so cleanenv falls back to the Go zero value.
	unsetEmptyEnvVars()

	err = cleanenv.ReadEnv(&env)
	if err != nil {
		log.Error("Configuration could not be loaded", "error", err)
		os.Exit(1)
	}

	if env.SMTPHost != "" && env.SMTPPort <= 0 {
		log.Error("SMTP_PORT must be a positive integer when SMTP_HOST is set", "value", env.SMTPPort)
		os.Exit(1)
	}

	// Set default value for ShowDbInstallationVerificationLogs if not defined
	if os.Getenv("SHOW_DB_INSTALLATION_VERIFICATION_LOGS") == "" {
		env.ShowDbInstallationVerificationLogs = true
	}

	// Set default value for IsCloud if not defined
	if os.Getenv("IS_CLOUD") == "" {
		env.IsCloud = false
	}

	for _, arg := range os.Args {
		if strings.Contains(arg, "test") {
			env.IsTesting = true
			break
		}
	}

	// Check for external database override
	if externalDsn := os.Getenv("DANGEROUS_EXTERNAL_DATABASE_DSN"); externalDsn != "" {
		log.Warn(
			"Using DANGEROUS_EXTERNAL_DATABASE_DSN - connecting to external database instead of internal PostgreSQL",
		)
		env.DatabaseDsn = externalDsn
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

	tools.LogAndExitIfClientToolsBroken(log, env.ShowDbInstallationVerificationLogs)

	if env.NodeNetworkThroughputMBs == 0 {
		env.NodeNetworkThroughputMBs = 125 // 1 Gbit/s
	}

	if !env.IsManyNodesMode {
		env.IsPrimaryNode = true
		env.IsProcessingNode = true
	}

	if env.TestLocalhost == "" {
		env.TestLocalhost = "localhost"
	}

	// Valkey
	if env.ValkeyHost == "" {
		log.Error("VALKEY_HOST is empty")
		os.Exit(1)
	}
	if env.ValkeyPort == "" {
		log.Error("VALKEY_PORT is empty")
		os.Exit(1)
	}

	// Check for external Valkey override
	if externalValkeyHost := os.Getenv("DANGEROUS_VALKEY_HOST"); externalValkeyHost != "" {
		log.Warn(
			"Using DANGEROUS_VALKEY_* variables - connecting to external Valkey instead of internal instance",
		)
		env.ValkeyHost = externalValkeyHost

		if externalValkeyPort := os.Getenv("DANGEROUS_VALKEY_PORT"); externalValkeyPort != "" {
			env.ValkeyPort = externalValkeyPort
		}
		if externalValkeyUsername := os.Getenv("DANGEROUS_VALKEY_USERNAME"); externalValkeyUsername != "" {
			env.ValkeyUsername = externalValkeyUsername
		}
		if externalValkeyPassword := os.Getenv("DANGEROUS_VALKEY_PASSWORD"); externalValkeyPassword != "" {
			env.ValkeyPassword = externalValkeyPassword
		}
		if externalValkeyIsSsl := os.Getenv("DANGEROUS_VALKEY_IS_SSL"); externalValkeyIsSsl != "" {
			env.ValkeyIsSsl = externalValkeyIsSsl == "true"
		}
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
		if env.TestPostgres12Port == "" {
			log.Error("TEST_POSTGRES_12_PORT is empty")
			os.Exit(1)
		}
		if env.TestPostgres13Port == "" {
			log.Error("TEST_POSTGRES_13_PORT is empty")
			os.Exit(1)
		}
		if env.TestPostgres14Port == "" {
			log.Error("TEST_POSTGRES_14_PORT is empty")
			os.Exit(1)
		}
		if env.TestPostgres15Port == "" {
			log.Error("TEST_POSTGRES_15_PORT is empty")
			os.Exit(1)
		}
		if env.TestPostgres16Port == "" {
			log.Error("TEST_POSTGRES_16_PORT is empty")
			os.Exit(1)
		}
		if env.TestPostgres17Port == "" {
			log.Error("TEST_POSTGRES_17_PORT is empty")
			os.Exit(1)
		}
		if env.TestPostgres18Port == "" {
			log.Error("TEST_POSTGRES_18_PORT is empty")
			os.Exit(1)
		}

		if env.TestMinioPort == "" {
			log.Error("TEST_MINIO_PORT is empty")
			os.Exit(1)
		}
		if env.TestMinioConsolePort == "" {
			log.Error("TEST_MINIO_CONSOLE_PORT is empty")
			os.Exit(1)
		}

		if env.TestAzuriteBlobPort == "" {
			log.Error("TEST_AZURITE_BLOB_PORT is empty")
			os.Exit(1)
		}

		if env.TestNASPort == "" {
			log.Error("TEST_NAS_PORT is empty")
			os.Exit(1)
		}

	}

	// Billing
	if env.IsCloud {
		if env.PricePerGBCents <= 0 {
			log.Error("PRICE_PER_GB_CENTS must be a positive integer in cloud mode", "value", env.PricePerGBCents)
			os.Exit(1)
		}

		if env.PaddleApiKey == "" {
			log.Error("PADDLE_API_KEY is empty")
			os.Exit(1)
		}

		if env.PaddleWebhookSecret == "" {
			log.Error("PADDLE_WEBHOOK_SECRET is empty")
			os.Exit(1)
		}

		if env.PaddlePriceID == "" {
			log.Error("PADDLE_PRICE_ID is empty")
			os.Exit(1)
		}

		if env.PaddleClientToken == "" {
			log.Error("PADDLE_CLIENT_TOKEN is empty")
			os.Exit(1)
		}
	}

	env.MinStorageGB = 20
	env.MaxStorageGB = 10_000
	env.TrialDuration = 24 * time.Hour
	env.TrialStorageGB = 20
	env.GracePeriod = 30 * 24 * time.Hour

	log.Info("Environment variables loaded successfully!")
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

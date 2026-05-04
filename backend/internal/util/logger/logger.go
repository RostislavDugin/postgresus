package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var (
	loggerInstance     *slog.Logger
	victoriaLogsWriter *VictoriaLogsWriter
)

var initLogger = sync.OnceFunc(func() {
	// Create stdout handler
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format("2006/01/02 15:04:05"))
			}
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return a
		},
	})

	// Try to initialize VictoriaLogs writer if configured
	// Note: This will be called before config is fully loaded in some cases,
	// so we need to handle that gracefully
	victoriaLogsWriter = tryInitVictoriaLogs()

	// Create multi-handler
	multiHandler := NewMultiHandler(stdoutHandler, victoriaLogsWriter)
	loggerInstance = slog.New(multiHandler)

	loggerInstance.Info("Text structured logger initialized")
	if victoriaLogsWriter != nil {
		loggerInstance.Info("VictoriaLogs enabled")
	} else {
		loggerInstance.Info("VictoriaLogs disabled")
	}
})

// GetLogger returns a singleton slog.Logger that logs to the console
func GetLogger() *slog.Logger {
	initLogger()
	return loggerInstance
}

// ShutdownVictoriaLogs gracefully shuts down the VictoriaLogs writer
var ShutdownVictoriaLogs = sync.OnceFunc(func() {
	if victoriaLogsWriter != nil {
		victoriaLogsWriter.Shutdown(5 * time.Second)
	}
})

func tryInitVictoriaLogs() *VictoriaLogsWriter {
	// Ensure .env is loaded before reading environment variables
	ensureEnvLoaded()

	// Try to get config - this may fail early in startup
	url := getVictoriaLogsURL()
	username := getVictoriaLogsUsername()
	password := getVictoriaLogsPassword()

	if url == "" {
		fmt.Println("VictoriaLogs URL is not set")
		return nil
	}

	return NewVictoriaLogsWriter(url, username, password)
}

var ensureEnvLoaded = sync.OnceFunc(func() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Warning: could not get current working directory: %v\n", err)
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

	if err := godotenv.Load(envPath); err == nil {
		fmt.Printf("Logger: loaded .env from %s\n", envPath)
		return
	}

	fmt.Println("Logger: .env file not found at repo root, using existing environment variables")
})

func getVictoriaLogsURL() string {
	return os.Getenv("VICTORIA_LOGS_URL")
}

func getVictoriaLogsUsername() string {
	return os.Getenv("VICTORIA_LOGS_USERNAME")
}

func getVictoriaLogsPassword() string {
	return os.Getenv("VICTORIA_LOGS_PASSWORD")
}

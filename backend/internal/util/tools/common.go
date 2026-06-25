package tools

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ToolCheckResult is one DB-version bundle's verification outcome. A bundle
// is considered fatal when missing/broken should crash the app at startup
// (Postgres) and non-fatal otherwise (MySQL, MariaDB, MongoDB).
type ToolCheckResult struct {
	Db      string
	Version string
	BinDir  string
	Errors  []error
	IsFatal bool
}

// checkBinDir verifies binDir exists and that every command in
// requiredCommands is present and has the executable bit set. Returns the
// collected errors. If the directory does not exist, returns a single
// not-found error so callers can treat it differently for fatal vs non-fatal
// bundles.
func checkBinDir(binDir string, requiredCommands []string) []error {
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		return []error{fmt.Errorf("client tools bin directory not found: %s", binDir)}
	}

	var errs []error
	for _, cmd := range requiredCommands {
		cmdPath := filepath.Join(binDir, cmd)

		info, err := os.Stat(cmdPath)
		if os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("client command not found: %s", cmdPath))
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("cannot stat %s: %w", cmdPath, err))
			continue
		}

		if info.Mode().Perm()&0o111 == 0 {
			errs = append(errs, fmt.Errorf("client command not executable: %s", cmdPath))
		}
	}

	return errs
}

// CheckAllClientTools verifies every expected client binary for the current
// arch is present and executable. Pure: never logs, never exits. Used by
// startup (config) and runtime (healthcheck).
func CheckAllClientTools() []ToolCheckResult {
	results := checkPostgresql()
	results = append(results, checkMysql()...)
	results = append(results, checkMariadb()...)
	results = append(results, checkMongodb()...)

	return results
}

// LogAndExitIfClientToolsBroken logs every check result and exits the process
// if any fatal-tier (Postgres) bundle has errors. Non-fatal bundles only warn.
// Used by config at application startup.
func LogAndExitIfClientToolsBroken(logger *slog.Logger, isShowLogs bool) {
	results := CheckAllClientTools()

	hasFatalError := false
	for _, r := range results {
		log := logger.With("db", r.Db, "version", r.Version, "path", r.BinDir)

		if len(r.Errors) == 0 {
			if isShowLogs {
				log.Info("client tools verified")
			}

			continue
		}

		for _, err := range r.Errors {
			if r.IsFatal {
				log.Error("client tools check failed", "error", err)
				hasFatalError = true
			} else {
				log.Warn("client tools check failed - support disabled", "error", err)
			}
		}
	}

	if hasFatalError {
		os.Exit(1)
	}
}

// ClientToolsHealthError returns an aggregated error if any fatal-tier bundle
// has check failures, else nil. Used by the healthcheck endpoint.
func ClientToolsHealthError() error {
	results := CheckAllClientTools()

	var msgs []string
	for _, r := range results {
		if !r.IsFatal {
			continue
		}

		for _, err := range r.Errors {
			msgs = append(msgs, fmt.Sprintf("%s %s: %s", r.Db, r.Version, err.Error()))
		}
	}

	if len(msgs) == 0 {
		return nil
	}

	return errors.New("client tools broken: " + strings.Join(msgs, "; "))
}

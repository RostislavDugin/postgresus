package tools

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// writeFakeClient puts an executable script in dir that prints output when run
// with any argument, so the check can be exercised without a real bundle.
func writeFakeClient(t *testing.T, dir, name, output string) {
	t.Helper()

	script := "#!/bin/sh\necho " + strconv.Quote(output) + "\n"
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake client %s: %v", path, err)
	}
}

func Test_RunClients_WhenTheClientReportsItsOwnLine_ReportsNoErrors(t *testing.T) {
	binDir := t.TempDir()
	writeFakeClient(t, binDir, "mariadb-dump",
		"mariadb-dump  Ver 10.19 Distrib 10.6.28-MariaDB, for debian-linux-gnu (x86_64)")

	errs := runClients(binDir, []string{"mariadb-dump"}, "10.6", parseMariadbClientVersion)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func Test_RunClients_WhenTheBundleHoldsAForeignVersion_ReportsAFailure(t *testing.T) {
	binDir := t.TempDir()
	// The state assets/tools/arm/mariadb/mariadb-10.6/ was in: a 10.11 binary
	// under a directory named for the 10.6 line.
	writeFakeClient(t, binDir, "mariadb-dump",
		"mariadb-dump  Ver 10.19 Distrib 10.11.14-MariaDB, for debian-linux-gnu (aarch64)")

	errs := runClients(binDir, []string{"mariadb-dump"}, "10.6", parseMariadbClientVersion)
	if len(errs) != 1 {
		t.Fatalf("expected one error, got %v", errs)
	}

	message := errs[0].Error()
	if !strings.Contains(message, "10.11.14") || !strings.Contains(message, "10.6") {
		t.Errorf("error should name the version found and the line expected, got %q", message)
	}
}

func Test_RunClients_WhenTheClientCannotStart_ReportsAFailure(t *testing.T) {
	binDir := t.TempDir()
	path := filepath.Join(binDir, "mysqldump")
	// An ELF header this host cannot load stands in for a missing shared
	// library: the file is there and executable, and still cannot run.
	if err := os.WriteFile(path, []byte("\x7fELF not a loadable image"), 0o755); err != nil {
		t.Fatalf("failed to write the unusable client: %v", err)
	}

	errs := runClients(binDir, []string{"mysqldump"}, "8.0", parseMysqlClientVersion)
	if len(errs) != 1 {
		t.Fatalf("expected one error, got %v", errs)
	}

	if !strings.Contains(errs[0].Error(), "cannot run") {
		t.Errorf("error should say the client cannot run, got %q", errs[0].Error())
	}
}

func Test_RunClients_WithNoReleaseLineToHoldTheBundleTo_OnlyRequiresTheClientToStart(t *testing.T) {
	binDir := t.TempDir()
	writeFakeClient(t, binDir, "mongodump", "mongodump version: 100.19.1")

	errs := runClients(binDir, []string{"mongodump"}, "", nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func Test_CheckAllClientTools_AgainstTheShippedBundles_ReportsEveryBundleVerified(t *testing.T) {
	for _, result := range CheckAllClientTools() {
		if len(result.Errors) == 0 {
			continue
		}

		// arm-only gaps cannot appear here: the test runs on the host's arch.
		t.Errorf("%s %s (%s): %v", result.Db, result.Version, result.BinDir, result.Errors)
	}
}

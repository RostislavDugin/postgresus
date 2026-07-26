package usecases_logical_mariadb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	compressionProbeTimeout = 15 * time.Second
	compressionProbeTable   = "__databasus_compression_probe__"
)

// networkCompressionCandidates has only the two entries MariaDB's protocol offers - zlib and
// nothing. Managed providers that front the server with a proxy advertise compression the proxy
// cannot actually negotiate, so the choice has to come from a handshake rather than be assumed.
var networkCompressionCandidates = [][]string{
	{"--compress"},
	{},
}

// MyCnfFile holds the credentials and TLS mode, so the probe must not re-derive them; IsHttps only
// adds the cert-verification opt-out that has no .my.cnf equivalent here.
type CompressionProbeSpec struct {
	MariadbDumpBin string
	MyCnfFile      string
	DatabaseName   string
	DatabaseID     uuid.UUID
	IsHttps        bool
}

func getNetworkCompressionLabel(candidate []string) string {
	if len(candidate) == 0 {
		return "uncompressed"
	}

	return strings.Join(candidate, " ")
}

// Error 2066 is CR_COMPRESSION_WRONGLY_CONFIGURED, raised client-side when the endpoint cannot
// negotiate the requested algorithm.
func isCompressionRejection(stderrStr string) bool {
	return containsIgnoreCase(stderrStr, "compression algorithm") ||
		containsIgnoreCase(stderrStr, "2066")
}

func (uc *CreateMariadbBackupUsecase) probeNetworkCompressionArgs(
	ctx context.Context,
	probe CompressionProbeSpec,
) []string {
	compressedCandidates := networkCompressionCandidates[:len(networkCompressionCandidates)-1]
	chosenCompressionArgs := networkCompressionCandidates[len(networkCompressionCandidates)-1]

	for _, candidate := range compressedCandidates {
		if uc.isNetworkCompressionAccepted(ctx, probe, candidate) {
			chosenCompressionArgs = candidate
			break
		}
	}

	uc.logger.Info(
		fmt.Sprintf("negotiated MariaDB network compression: %s",
			getNetworkCompressionLabel(chosenCompressionArgs)),
		"database_id", probe.DatabaseID,
	)

	return chosenCompressionArgs
}

// isNetworkCompressionAccepted dials with mariadb-dump rather than the interactive client so the
// probe cannot disagree with the dump about which binary loads. Naming a table that cannot exist
// keeps the attempt O(1): the handshake — where compression is negotiated — happens first, then
// mariadb-dump gives up. Only a compression complaint counts as a rejection; any other failure
// (unreachable host, bad credentials) recurs identically for every candidate, so it is left for
// the dump to report with a precise message.
func (uc *CreateMariadbBackupUsecase) isNetworkCompressionAccepted(
	ctx context.Context,
	probe CompressionProbeSpec,
	candidate []string,
) bool {
	probeCtx, cancel := context.WithTimeout(ctx, compressionProbeTimeout)
	defer cancel()

	args := append([]string{"--defaults-file=" + probe.MyCnfFile}, candidate...)

	if probe.IsHttps {
		args = append(args, "--skip-ssl-verify-server-cert")
	}

	args = append(args, "--no-data", probe.DatabaseName, compressionProbeTable)

	var probeStderr bytes.Buffer

	cmd := exec.CommandContext(probeCtx, probe.MariadbDumpBin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = &probeStderr
	cmd.Env = append(os.Environ(),
		"MYSQL_PWD=",
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	if err := cmd.Run(); err != nil && isCompressionRejection(probeStderr.String()) {
		uc.logger.Debug(
			fmt.Sprintf("MariaDB rejected network compression: %s",
				getNetworkCompressionLabel(candidate)),
			"database_id", probe.DatabaseID,
			"stderr", probeStderr.String(),
		)

		return false
	}

	return true
}

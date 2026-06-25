package restore

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const ignoredErrorsMarker = "errors ignored on restore:"

var extensionNamePattern = regexp.MustCompile(`extension "([^"]+)"`)

// IsMissingExtensionOnly reports whether a non-zero pg_restore exit is explained
// solely by extensions absent from this environment — i.e. the data itself
// restored. pg_restore runs without --exit-on-error, so it tails "errors ignored
// on restore: N"; this returns true only when every visible item error is
// extension-related and their count equals N. The count guard matters because
// StderrTail is capped at 8192 bytes: a truncated tail could hide a non-extension
// error, so when N can't be fully accounted for we refuse to tolerate.
func IsMissingExtensionOnly(stderrTail string) bool {
	ignoredCount, hasMarker := parseIgnoredErrorCount(stderrTail)
	if !hasMarker || ignoredCount < 1 {
		return false
	}

	itemErrors := queryErrorLines(stderrTail)
	if len(itemErrors) != ignoredCount {
		return false
	}

	for _, line := range itemErrors {
		if !isExtensionErrorLine(line) {
			return false
		}
	}

	return true
}

// ExtractUnavailableExtensions returns the de-duplicated, sorted names of the
// extensions pg_restore could not create, for an operator-facing log line.
func ExtractUnavailableExtensions(stderrTail string) []string {
	seen := map[string]struct{}{}

	var names []string

	for _, line := range queryErrorLines(stderrTail) {
		if !isExtensionErrorLine(line) {
			continue
		}

		for _, match := range extensionNamePattern.FindAllStringSubmatch(line, -1) {
			name := match[1]
			if _, isDup := seen[name]; isDup {
				continue
			}

			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	slices.Sort(names)

	return names
}

func parseIgnoredErrorCount(stderrTail string) (int, bool) {
	idx := strings.LastIndex(strings.ToLower(stderrTail), ignoredErrorsMarker)
	if idx < 0 {
		return 0, false
	}

	fields := strings.Fields(stderrTail[idx+len(ignoredErrorsMarker):])
	if len(fields) == 0 {
		return 0, false
	}

	count, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, false
	}

	return count, true
}

func queryErrorLines(stderrTail string) []string {
	var lines []string

	for line := range strings.SplitSeq(stderrTail, "\n") {
		if strings.Contains(strings.ToLower(line), "could not execute query: error:") {
			lines = append(lines, line)
		}
	}

	return lines
}

// isExtensionErrorLine matches the phrasings a missing extension produces — the
// failed CREATE EXTENSION and its cascading COMMENT ON EXTENSION. The "extension"
// guard keeps unrelated cascades (`type "..." does not exist`) out.
func isExtensionErrorLine(line string) bool {
	lowered := strings.ToLower(line)
	if !strings.Contains(lowered, "extension") {
		return false
	}

	return strings.Contains(lowered, "is not available") ||
		strings.Contains(lowered, "could not open extension control file") ||
		strings.Contains(lowered, "does not exist")
}

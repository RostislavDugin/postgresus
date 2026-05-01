package clickhouse

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// DDL transformations are split into three exported functions, all backed by
// the same balanced-paren / string-literal-aware scanner:
//
//   CleanDDLForBackup            — backup-time, strips ON CLUSTER and UUID '...'
//   RewriteEngineForRestore      — restore-time, drops Replicated/Shared engine
//                                  prefix + the path/replica args, gated on
//                                  the target database's keepReplicated flag
//   RewriteIdentifiersForRestore — restore-time, substitutes the source database
//                                  qualifier with the target name
//
// The scanner is intentionally hand-written rather than using a real SQL
// parser: ClickHouse syntax evolves quickly (Variant, Dynamic, JSON, etc.) and
// a parser that doesn't track upstream perfectly is a worse silent-corruption
// risk than an unknown-input fail-closed return.

// CleanDDLForBackup strips ON CLUSTER and UUID '...' clauses from a CREATE
// statement. The engine clause is preserved verbatim — Replicated*MergeTree
// and Shared*MergeTree pass through unchanged. Returns warnings for anything
// the scanner removed (so callers can log non-trivial transformations).
func CleanDDLForBackup(ddl string) (string, []string, error) {
	out, w1, err := stripOnCluster(ddl)
	if err != nil {
		return "", nil, fmt.Errorf("strip ON CLUSTER: %w", err)
	}

	out, w2, err := stripUUID(out)
	if err != nil {
		return "", nil, fmt.Errorf("strip UUID: %w", err)
	}

	return out, append(w1, w2...), nil
}

// RewriteEngineForRestore rewrites the ENGINE clause to drop Replicated and
// Shared prefixes (and their first two ZooKeeper-coordination args) when
// keepReplicated is false. CREATE DATABASE ENGINE = Replicated(...) collapses
// to ENGINE = Atomic. When keepReplicated is true, the input is returned
// verbatim with no warnings.
func RewriteEngineForRestore(cleanedDdl string, keepReplicated bool) (string, []string, error) {
	if keepReplicated {
		return cleanedDdl, nil, nil
	}

	enginePos, err := findEngineEqualsKeyword(cleanedDdl)
	if err != nil {
		return "", nil, err
	}
	if enginePos < 0 {
		// No ENGINE clause (rare but possible for CREATE DATABASE without an
		// explicit engine — ClickHouse defaults to Atomic). Nothing to rewrite.
		return cleanedDdl, nil, nil
	}

	nameStart := skipSpaces(cleanedDdl, enginePos)
	nameEnd := scanIdentifier(cleanedDdl, nameStart)
	if nameEnd == nameStart {
		return "", nil, errors.New("ENGINE = is not followed by an identifier")
	}

	engineName := cleanedDdl[nameStart:nameEnd]
	argsStart, argsEnd, hasArgs, err := scanEngineArgs(cleanedDdl, nameEnd)
	if err != nil {
		return "", nil, err
	}

	isCreateDatabase := isCreateDatabaseStatement(cleanedDdl)
	rewritten, warnings, err := rewriteEngineClause(engineName, cleanedDdl, nameStart, nameEnd, argsStart, argsEnd, hasArgs, isCreateDatabase)
	if err != nil {
		return "", nil, err
	}

	return rewritten, warnings, nil
}

// RewriteIdentifiersForRestore replaces every `<sourceDb>.<tbl>` qualifier with
// `<targetDb>.<tbl>`, applying its own backquote-quoting. Both arguments must
// be UNQUOTED database names. Identifiers that appear inside single-quoted
// string literals are left untouched (a documented v1 limitation: an MV's
// as_select referencing the source DB name in a string literal must be edited
// manually).
func RewriteIdentifiersForRestore(ddl, sourceDb, targetDb string) (string, error) {
	if sourceDb == "" {
		return "", errors.New("sourceDb is required")
	}
	if targetDb == "" {
		return "", errors.New("targetDb is required")
	}
	if sourceDb == targetDb {
		return ddl, nil
	}

	var b strings.Builder
	b.Grow(len(ddl) + 32)

	quotedTarget := quoteIdent(targetDb)

	// We accept the source identifier in three forms:
	//   `srcDb`         — backquoted, possibly with embedded `` for backtick
	//   "srcDb"         — double-quoted (less common in CH but accepted)
	//   srcDb           — bare (must be a complete identifier token)
	srcBacktick := quoteIdent(sourceDb)
	srcDoubleQuoted := `"` + strings.ReplaceAll(sourceDb, `"`, `""`) + `"`

	i := 0
	for i < len(ddl) {
		ch := ddl[i]

		// Pass through string literals verbatim.
		if ch == '\'' {
			end := skipSingleQuotedString(ddl, i)
			b.WriteString(ddl[i:end])
			i = end
			continue
		}

		// Try to recognise <sourceDb>. as a database-qualifier prefix.
		if matched := matchSourceQualifier(ddl, i, sourceDb, srcBacktick, srcDoubleQuoted); matched > 0 {
			// matched is the byte length of the source identifier (including
			// any quoting). The next byte must be '.' to count as a qualifier
			// — we handle that here so a coincidental occurrence of the db name
			// outside a qualifier (e.g. as a column alias) is untouched.
			next := i + matched
			if next < len(ddl) && ddl[next] == '.' {
				b.WriteString(quotedTarget)
				b.WriteByte('.')
				i = next + 1
				continue
			}
		}

		b.WriteByte(ch)
		i++
	}

	return b.String(), nil
}

// --- Internal helpers ---

// stripOnCluster removes the ON CLUSTER <name> clause (and its leading whitespace).
// <name> may be backquoted, single-quoted, or a bare identifier. The clause is
// matched case-insensitively, only outside string literals.
func stripOnCluster(ddl string) (string, []string, error) {
	idx := indexOfKeyword(ddl, "ON CLUSTER", 0)
	if idx < 0 {
		return ddl, nil, nil
	}

	clauseStart := idx
	if clauseStart > 0 && isSpace(ddl[clauseStart-1]) {
		clauseStart--
	}

	end := idx + len("ON CLUSTER")
	end = skipSpaces(ddl, end)

	// The cluster name is one of: '...', "...", `...`, or a bare identifier.
	if end >= len(ddl) {
		return "", nil, errors.New("ON CLUSTER missing cluster name")
	}

	switch ddl[end] {
	case '\'':
		end = skipSingleQuotedString(ddl, end)
	case '`':
		end = skipBacktickIdentifier(ddl, end)
	case '"':
		end = skipDoubleQuotedIdentifier(ddl, end)
	default:
		end = scanIdentifier(ddl, end)
	}

	cleaned := ddl[:clauseStart] + ddl[end:]
	rest, w, err := stripOnCluster(cleaned)
	if err != nil {
		return "", nil, err
	}
	return rest, append([]string{"stripped ON CLUSTER clause"}, w...), nil
}

// stripUUID removes a UUID '<uuid>' clause. ClickHouse only emits this in
// SHOW CREATE TABLE output for Atomic databases; reusing the source UUID at
// restore time can collide with whatever the target server assigns.
func stripUUID(ddl string) (string, []string, error) {
	idx := indexOfKeyword(ddl, "UUID", 0)
	if idx < 0 {
		return ddl, nil, nil
	}

	// UUID can also legitimately appear as a column TYPE keyword. We require
	// the next non-space token to be a single-quoted string literal — column
	// type usage is followed by a comma, paren, or NULL/DEFAULT/etc., never a
	// quoted literal directly.
	clauseStart := idx
	if clauseStart > 0 && isSpace(ddl[clauseStart-1]) {
		clauseStart--
	}

	end := idx + len("UUID")
	skipped := skipSpaces(ddl, end)
	if skipped >= len(ddl) || ddl[skipped] != '\'' {
		// Not a UUID '...' clause; advance past this occurrence and recurse.
		rest, w, err := stripUUID(ddl[idx+len("UUID"):])
		if err != nil {
			return "", nil, err
		}
		return ddl[:idx+len("UUID")] + rest, w, nil
	}

	end = skipSingleQuotedString(ddl, skipped)

	cleaned := ddl[:clauseStart] + ddl[end:]
	rest, w, err := stripUUID(cleaned)
	if err != nil {
		return "", nil, err
	}
	return rest, append([]string{"stripped UUID clause"}, w...), nil
}

// findEngineEqualsKeyword locates `ENGINE =` (or `ENGINE=`) and returns the
// byte position immediately after the `=`. Returns -1 if no ENGINE clause.
//
// A naive search that returns the first `ENGINE` keyword fails open: a column
// named `engine` (e.g. `` `engine` String ``) is matched before the real
// engine clause, which would silently leave Replicated*MergeTree DDL
// untouched. We loop past keyword hits that are not immediately followed by
// `=` until we find the real one or run out of input.
func findEngineEqualsKeyword(ddl string) (int, error) {
	start := 0
	for {
		pos := indexOfKeyword(ddl, "ENGINE", start)
		if pos < 0 {
			return -1, nil
		}

		after := pos + len("ENGINE")
		afterSkipped := skipSpaces(ddl, after)
		if afterSkipped < len(ddl) && ddl[afterSkipped] == '=' {
			return afterSkipped + 1, nil
		}

		start = pos + len("ENGINE")
	}
}

// scanEngineArgs handles three cases:
//   - ENGINE = Foo(args)  — argsStart points just after '(', argsEnd at ')'
//   - ENGINE = Foo()      — empty arg list, hasArgs=true
//   - ENGINE = Foo        — no parens at all, hasArgs=false
func scanEngineArgs(ddl string, after int) (argsStart, argsEnd int, hasArgs bool, err error) {
	skipped := skipSpaces(ddl, after)
	if skipped >= len(ddl) || ddl[skipped] != '(' {
		return after, after, false, nil
	}

	open := skipped
	close, err := findMatchingParen(ddl, open)
	if err != nil {
		return 0, 0, false, err
	}

	return open + 1, close, true, nil
}

func rewriteEngineClause(
	engineName, ddl string,
	nameStart, nameEnd, argsStart, argsEnd int, hasArgs, isCreateDatabase bool,
) (string, []string, error) {
	// CREATE DATABASE ENGINE = Replicated(...) → Atomic
	if isCreateDatabase && engineName == "Replicated" {
		end := nameEnd
		if hasArgs {
			end = argsEnd + 1
		}
		return ddl[:nameStart] + "Atomic" + ddl[end:], []string{"rewrote CREATE DATABASE ENGINE = Replicated(...) to Atomic"}, nil
	}

	switch {
	case strings.HasPrefix(engineName, "Replicated"):
		return rewriteEngineDropReplicatedPrefix(engineName, ddl, nameStart, nameEnd, argsStart, argsEnd, hasArgs)
	case strings.HasPrefix(engineName, "Shared"):
		stripped := strings.TrimPrefix(engineName, "Shared")
		if stripped == "" {
			return "", nil, fmt.Errorf("unexpected engine name %q", engineName)
		}
		return ddl[:nameStart] + stripped + ddl[nameEnd:], []string{fmt.Sprintf("rewrote engine %s → %s", engineName, stripped)}, nil
	}

	return ddl, nil, nil
}

func rewriteEngineDropReplicatedPrefix(
	engineName, ddl string,
	nameStart, nameEnd, argsStart, argsEnd int, hasArgs bool,
) (string, []string, error) {
	stripped := strings.TrimPrefix(engineName, "Replicated")
	if stripped == "" {
		return "", nil, fmt.Errorf("unexpected engine name %q (Replicated alone)", engineName)
	}

	if !hasArgs {
		return ddl[:nameStart] + stripped + ddl[nameEnd:], []string{fmt.Sprintf("rewrote engine %s → %s", engineName, stripped)}, nil
	}

	// Replicated*MergeTree's first two args are the ZK path and replica name.
	// We drop them and keep everything after the second top-level comma.
	innerStart := argsStart
	innerEnd := argsEnd
	args := ddl[innerStart:innerEnd]

	commas, err := topLevelCommas(args)
	if err != nil {
		return "", nil, err
	}

	var newArgs string
	switch {
	case len(commas) >= 2:
		// Keep everything after the second comma (and skip the comma + leading spaces).
		rest := strings.TrimLeft(args[commas[1]+1:], " \t\n")
		newArgs = rest
	default:
		// Fewer than 2 args present — must mean only path or path+replica with
		// no engine-specific tail. Drop everything.
		newArgs = ""
	}

	prefix := ddl[:nameStart] + stripped + "("
	suffix := ")" + ddl[innerEnd+1:]
	return prefix + newArgs + suffix,
		[]string{fmt.Sprintf("rewrote engine %s(...) → %s(...) and dropped 2 replication args", engineName, stripped)},
		nil
}

func isCreateDatabaseStatement(ddl string) bool {
	createIdx := indexOfKeyword(ddl, "CREATE", 0)
	if createIdx < 0 {
		return false
	}

	databaseIdx := indexOfKeyword(ddl, "DATABASE", createIdx+len("CREATE"))
	if databaseIdx < 0 {
		return false
	}

	// Anything else (TABLE, MATERIALIZED VIEW, etc.) appearing first means
	// this is not a CREATE DATABASE.
	for _, kw := range []string{"TABLE", "VIEW", "DICTIONARY", "FUNCTION"} {
		idx := indexOfKeyword(ddl, kw, createIdx+len("CREATE"))
		if idx >= 0 && idx < databaseIdx {
			return false
		}
	}

	return true
}

// --- Low-level scanning primitives ---

// indexOfKeyword finds the first occurrence of kw (case-insensitive) where the
// match is bounded by non-identifier characters and lies outside any
// single-quoted string, backtick-quoted identifier, or double-quoted
// identifier. Returns -1 if not found.
//
// Quoted-identifier handling is critical: a column named `` `engine` `` would
// otherwise match as if it were the ENGINE keyword, and an `INSERT INTO
// "engine_logs" ...` statement would similarly false-match.
func indexOfKeyword(s, kw string, start int) int {
	upper := strings.ToUpper(kw)

	i := start
	for i < len(s) {
		switch s[i] {
		case '\'':
			i = skipSingleQuotedString(s, i)
			continue
		case '`':
			i = skipBacktickIdentifier(s, i)
			continue
		case '"':
			i = skipDoubleQuotedIdentifier(s, i)
			continue
		}
		if i+len(kw) > len(s) {
			return -1
		}
		// Boundary check on the left.
		if i > 0 && isIdentChar(s[i-1]) {
			i++
			continue
		}
		if strings.EqualFold(s[i:i+len(kw)], upper) {
			// Boundary check on the right.
			rb := i + len(kw)
			if rb < len(s) && isIdentChar(s[rb]) {
				i++
				continue
			}
			return i
		}
		i++
	}
	return -1
}

func skipSpaces(s string, pos int) int {
	for pos < len(s) && isSpace(s[pos]) {
		pos++
	}
	return pos
}

func scanIdentifier(s string, pos int) int {
	for pos < len(s) && isIdentChar(s[pos]) {
		pos++
	}
	return pos
}

// skipSingleQuotedString assumes s[start] == '\''. Returns position past the
// closing quote, handling '' and \' escapes. If unterminated, returns len(s).
func skipSingleQuotedString(s string, start int) int {
	i := start + 1
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if c == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(s)
}

func skipBacktickIdentifier(s string, start int) int {
	i := start + 1
	for i < len(s) {
		c := s[i]
		if c == '`' {
			if i+1 < len(s) && s[i+1] == '`' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(s)
}

func skipDoubleQuotedIdentifier(s string, start int) int {
	i := start + 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			if i+1 < len(s) && s[i+1] == '"' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return len(s)
}

// findMatchingParen assumes s[open] == '('. Returns the index of the matching
// ')', tracking nested parens and string literals.
func findMatchingParen(s string, open int) (int, error) {
	depth := 1
	i := open + 1
	for i < len(s) {
		switch s[i] {
		case '\'':
			i = skipSingleQuotedString(s, i)
			continue
		case '`':
			i = skipBacktickIdentifier(s, i)
			continue
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
		i++
	}
	return 0, errors.New("unbalanced parentheses")
}

// topLevelCommas returns the indices (relative to s) of every comma that lies
// at paren-depth 0, ignoring commas inside nested parens or string literals.
func topLevelCommas(s string) ([]int, error) {
	var out []int
	depth := 0
	i := 0
	for i < len(s) {
		switch s[i] {
		case '\'':
			i = skipSingleQuotedString(s, i)
			continue
		case '`':
			i = skipBacktickIdentifier(s, i)
			continue
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, errors.New("unbalanced parentheses in args")
			}
		case ',':
			if depth == 0 {
				out = append(out, i)
			}
		}
		i++
	}
	if depth != 0 {
		return nil, errors.New("unbalanced parentheses in args")
	}
	return out, nil
}

// matchSourceQualifier returns the byte length of the source-db identifier at
// position i (including any quoting), or 0 if no match. The caller is
// responsible for checking that the next byte is '.'.
func matchSourceQualifier(ddl string, i int, sourceDb, srcBacktick, srcDoubleQuoted string) int {
	if strings.HasPrefix(ddl[i:], srcBacktick) {
		return len(srcBacktick)
	}
	if strings.HasPrefix(ddl[i:], srcDoubleQuoted) {
		return len(srcDoubleQuoted)
	}
	if i > 0 && isIdentChar(ddl[i-1]) {
		return 0
	}
	if !strings.HasPrefix(ddl[i:], sourceDb) {
		return 0
	}
	end := i + len(sourceDb)
	if end < len(ddl) && isIdentChar(ddl[end]) {
		return 0
	}
	return len(sourceDb)
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isIdentChar(c byte) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	if c == '_' {
		return true
	}
	// Identifiers may contain $; ClickHouse also accepts unicode letters in
	// backtick-quoted identifiers, but those are matched via skipBacktickIdentifier.
	return c == '$'
}

// unicodeIsLetter is unused but kept here so callers can swap isIdentChar for
// a unicode-aware variant if test fixtures ever exercise it.
var _ = unicode.IsLetter

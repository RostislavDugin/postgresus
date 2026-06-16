package sqldumpfilter

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// TableSectionRe matches the comment headers in mysqldump/mariadb-dump output
// that mark the beginning of a per-table DDL or data block.
var TableSectionRe = regexp.MustCompile("^-- (?:Table structure for table|Dumping data for table) `([^`]+)`")

// NewTableFilterReader filters the SQL dump stream by table name.
// When includeTables is non-empty, only those tables are emitted (excludeTables is ignored).
// When only excludeTables is set, all tables except those are emitted.
// Returns r unchanged when both slices are empty.
func NewTableFilterReader(r io.Reader, includeTables, excludeTables []string) io.Reader {
	if len(includeTables) == 0 && len(excludeTables) == 0 {
		return r
	}

	includeSet := MakeTableSet(includeTables)
	excludeSet := MakeTableSet(excludeTables)

	pr, pw := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)

		shouldEmit := true

		for scanner.Scan() {
			line := scanner.Bytes()
			lineStr := string(line)

			if m := TableSectionRe.FindStringSubmatch(lineStr); m != nil {
				if len(includeSet) > 0 {
					shouldEmit = includeSet[m[1]]
				} else {
					shouldEmit = !excludeSet[m[1]]
				}
				if shouldEmit {
					if _, err := pw.Write(append(line, '\n')); err != nil {
						pw.CloseWithError(err)
						return
					}
				}
				continue
			}

			// After UNLOCK TABLES; the per-table section ends and shouldEmit is reset.
			// Any post-table SQL mysqldump appends (views, routines) is therefore always
			// emitted — stream-based filtering cannot suppress it without materializing
			// the full dump first.
			if lineStr == "UNLOCK TABLES;" {
				if shouldEmit {
					if _, err := pw.Write(append(line, '\n')); err != nil {
						pw.CloseWithError(err)
						return
					}
				}
				shouldEmit = true
				continue
			}

			if shouldEmit {
				if _, err := pw.Write(append(line, '\n')); err != nil {
					pw.CloseWithError(err)
					return
				}
			}
		}

		pw.CloseWithError(scanner.Err())
	}()

	return pr
}

// MakeTableSet builds a lookup set from a list of table names, trimming whitespace
// and dropping empty entries.
func MakeTableSet(tables []string) map[string]bool {
	set := make(map[string]bool, len(tables))
	for _, t := range tables {
		if t = strings.TrimSpace(t); t != "" {
			set[t] = true
		}
	}
	return set
}

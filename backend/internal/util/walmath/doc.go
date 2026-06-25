// Package walmath implements pure WAL primitives: LSN parsing, WAL segment
// filename arithmetic, and .history file parsing.
//
// Several files are adapted from WAL-G (github.com/wal-g/wal-g, Apache-2.0)
// — see per-file headers for upstream paths and copyright. Tests follow the
// project's Test_<What>_<Conditions>_<Expected> naming; substance of
// production-proven WAL-G edge cases is preserved.
package walmath

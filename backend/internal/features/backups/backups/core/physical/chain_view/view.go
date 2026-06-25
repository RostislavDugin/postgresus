package chain_view

import (
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/util/walmath"
)

// ChainView is one backup chain — its FULL, its INCRs and the WAL between.
// It represents a single chain from one FULL to the next FULL.
//
// LSN layout of the fields:
//
// Span:             [════════════════════════════════════════]
// WalSegments:      [seg1][seg2]        [seg4][seg5]
// Gaps:                         [══════]
// MaxReplayableLSN:            ↑ (end of seg2 — gap blocks further replay)
type ChainView struct {
	RootFull     *physical_models.PhysicalFullBackup
	Incrementals []*physical_models.PhysicalIncrementalBackup
	WalSegments  []*physical_models.PhysicalWalSegment

	// Span is the LSN range owned by this chain: from RootFull.StartLSN up to
	// the next FULL's StartLSN (or LSNMax if no next FULL exists yet). Only
	// FULLs bound a chain — INCRs sit inside Span, not at its edge.
	Span LSNRange

	// Gaps are holes between WalSegments inside Span — a missing segment or a
	// mid-chain timeline switch. The chain stays usable, but PITR into a gap
	// is impossible.
	Gaps []LSNRange

	// MaxReplayableLSN is the furthest LSN PITR can reach: RootFull.StopLSN
	// plus every contiguous WAL segment up to (but not past) the first Gap.
	MaxReplayableLSN walmath.LSN
}

// LSNMax is the unbounded upper sentinel for the latest chain on a timeline
// (no successor FULL exists). Encoded as the maximum uint64 LSN so range
// predicates against pg_lsn columns still work without a special case.
const LSNMax walmath.LSN = ^walmath.LSN(0)

type LSNRange struct {
	Start walmath.LSN
	End   walmath.LSN
}

func (r LSNRange) Contains(lsn walmath.LSN) bool {
	return lsn >= r.Start && lsn < r.End
}

func (r LSNRange) IsUnbounded() bool {
	return r.End == LSNMax
}

type ValidationResult struct {
	Status  ValidationStatus
	Message string
}

// WalOrphanRef points at a WAL segment the cleaner orphan pass should sweep.
type WalOrphanRef struct {
	WalSegment *physical_models.PhysicalWalSegment
}

package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/util/walmath"
)

// SummarizerDecision encodes the outcome of a per-tick incremental pre-check.
// The scheduler in PR 3 maps each value to one of: spawn INCR, wait then
// recheck, spawn FULL on same chain, or spawn FULL anchoring a new chain.
type SummarizerDecision int

const (
	DecisionGoIncremental SummarizerDecision = iota
	DecisionWait
	DecisionFullSameChain
	DecisionFullNewChain
)

// SummarizerResult carries the decision plus the inputs the caller needs to
// act on it: wait/poll cadence for DecisionWait, error_reason for the
// chain-killing DecisionFullNewChain branches.
type SummarizerResult struct {
	Decision  SummarizerDecision
	WaitFor   time.Duration
	PollEvery time.Duration
	Reason    *physical_enums.PhysicalBackupErrorReason
}

const (
	// summarizerWaitPollInterval — how often the scheduler in PR 3 will poll
	// after DecisionWait. Five seconds is tight enough that recovery is felt
	// quickly, loose enough that we don't hammer pg_available_wal_summaries.
	summarizerWaitPollInterval = 5 * time.Second

	// summarizerWaitCap — DecisionWait timeout is min(cadence/4, this).
	summarizerWaitCap = 30 * time.Minute

	// summarizerFullThresholdBytes — once the summarizer trails current WAL by
	// more than this, we stop waiting and spawn a FULL: a gap this large would
	// stall pg_basebackup --incremental for too long, so a fresh FULL is the
	// cheaper recovery. Generous on purpose — we'd rather wait out a temporary
	// summarizer backlog (DecisionWait) than rotate to FULL prematurely.
	summarizerFullThresholdBytes int64 = 1024 * 1024 * 1024

	// summarizerGoSegments — the summarizer never summarizes the currently
	// active segment (and may have one more in flight), so a trailing lag of a
	// couple of segments is the healthy steady state, not "behind". A lag below
	// summarizerGoSegments × wal_segment_size means "caught up": go incremental
	// directly and let pg_basebackup wait out the last sliver itself.
	summarizerGoSegments int64 = 2
)

// CheckSummarizerReadiness classifies the state of the WAL summarizer relative to
// prevStopLSN (the parent backup's stop_lsn, or — for the WAL-gap fallback
// path — the current LSN). The conn must be an ordinary, non-replication
// connection.
func CheckSummarizerReadiness(
	ctx context.Context,
	conn *pgx.Conn,
	prevStopLSN walmath.LSN,
	incrementalCadence time.Duration,
) (SummarizerResult, error) {
	enabled, err := isSummarizerEnabled(ctx, conn)
	if err != nil {
		return SummarizerResult{}, err
	}

	if !enabled {
		reason := physical_enums.PhysicalBackupErrorSummarizerOff

		return SummarizerResult{
			Decision: DecisionFullNewChain,
			Reason:   &reason,
		}, nil
	}

	window, err := readSummaryWindow(ctx, conn)
	if err != nil {
		return SummarizerResult{}, err
	}

	// A parent stop_lsn below the oldest retained summary has aged out for good — no
	// future summary will ever cover it, so the chain must re-anchor on a fresh FULL.
	if window.hasAny && prevStopLSN < window.oldestStart {
		reason := physical_enums.PhysicalBackupErrorSummariesExpired

		return SummarizerResult{
			Decision: DecisionFullNewChain,
			Reason:   &reason,
		}, nil
	}

	// Summarizer on but nothing produced yet: not expiry, just not ready. Wait for the
	// first summary rather than mismeasuring lag against a NULL MAX(end_lsn).
	if !window.hasAny {
		return SummarizerResult{
			Decision:  DecisionWait,
			WaitFor:   waitWindowForCadence(incrementalCadence),
			PollEvery: summarizerWaitPollInterval,
		}, nil
	}

	lag, err := measureSummarizerLag(ctx, conn)
	if err != nil {
		return SummarizerResult{}, err
	}

	if lag >= summarizerFullThresholdBytes {
		return SummarizerResult{Decision: DecisionFullSameChain}, nil
	}

	goThreshold, err := summarizerGoThresholdBytes(ctx, conn)
	if err != nil {
		return SummarizerResult{}, err
	}

	// A trailing lag within the active-segment band is the healthy steady state —
	// go incremental. prevStopLSN is at or above the oldest retained summary (the
	// aged-out case bailed above); if it sits in the last unsummarized sliver,
	// pg_basebackup --incremental waits that segment out itself.
	if lag < goThreshold {
		return SummarizerResult{Decision: DecisionGoIncremental}, nil
	}

	return SummarizerResult{
		Decision:  DecisionWait,
		WaitFor:   waitWindowForCadence(incrementalCadence),
		PollEvery: summarizerWaitPollInterval,
	}, nil
}

// summarizerGoThresholdBytes is the lag (in bytes) below which the summarizer
// counts as caught up: summarizerGoSegments × the cluster's wal_segment_size.
// wal_segment_size is a server GUC reported in bytes by current_setting.
func summarizerGoThresholdBytes(ctx context.Context, conn *pgx.Conn) (int64, error) {
	var segmentSizeBytes int64

	// current_setting reports the GUC with its display unit (e.g. "16MB"), so
	// run it through pg_size_bytes rather than a raw ::bigint cast.
	if err := conn.QueryRow(ctx,
		"SELECT pg_size_bytes(current_setting('wal_segment_size'))::bigint",
	).Scan(&segmentSizeBytes); err != nil {
		return 0, fmt.Errorf("read wal_segment_size: %w", err)
	}

	return summarizerGoSegments * segmentSizeBytes, nil
}

// summarizerCheck is the readiness probe behind a package var so the
// bounded-wait loop can be unit-tested without a live summarizer.
var summarizerCheck = CheckSummarizerReadiness

// resolveSummarizerDecision runs the readiness probe and, when it reports
// DecisionWait (summarizer on and covering, but trailing current WAL), polls
// until the summarizer catches up, falls definitively behind, or the bounded
// window elapses. The returned decision is never DecisionWait: a window that
// expires while still lagging collapses to DecisionFullSameChain so the caller
// closes the chain rather than racing a moving target.
func resolveSummarizerDecision(
	ctx context.Context,
	conn *pgx.Conn,
	prevStopLSN walmath.LSN,
	cadence time.Duration,
) (SummarizerResult, error) {
	result, err := summarizerCheck(ctx, conn, prevStopLSN, cadence)
	if err != nil {
		return SummarizerResult{}, err
	}

	if result.Decision != DecisionWait {
		return result, nil
	}

	return waitForSummarizer(ctx, conn, prevStopLSN, cadence, result)
}

// waitForSummarizer polls the readiness probe on result.PollEvery until the
// summarizer reaches a terminal decision or result.WaitFor elapses. It never
// writes any catalog state — the INCR row stays IN_PROGRESS throughout, so a
// recovery within the window proceeds to a normal incremental with no
// intermediate CHAIN_BROKEN.
func waitForSummarizer(
	ctx context.Context,
	conn *pgx.Conn,
	prevStopLSN walmath.LSN,
	cadence time.Duration,
	initial SummarizerResult,
) (SummarizerResult, error) {
	deadline := time.Now().UTC().Add(initial.WaitFor)

	ticker := time.NewTicker(initial.PollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return SummarizerResult{}, ctx.Err()

		case <-ticker.C:
			result, err := summarizerCheck(ctx, conn, prevStopLSN, cadence)
			if err != nil {
				return SummarizerResult{}, err
			}

			if result.Decision != DecisionWait {
				return result, nil
			}

			if time.Now().UTC().After(deadline) {
				return SummarizerResult{Decision: DecisionFullSameChain}, nil
			}
		}
	}
}

func isSummarizerEnabled(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var setting string

	if err := conn.QueryRow(ctx,
		"SELECT setting FROM pg_settings WHERE name = 'summarize_wal'",
	).Scan(&setting); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("read summarize_wal setting: %w", err)
	}

	return setting == "on", nil
}

type summaryWindow struct {
	// hasAny is false when the summarizer is on but has produced no summary file yet —
	// just enabled, or idle before the first checkpoint.
	hasAny      bool
	oldestStart walmath.LSN
	newestEnd   walmath.LSN
}

func readSummaryWindow(ctx context.Context, conn *pgx.Conn) (summaryWindow, error) {
	var oldestStart, newestEnd *string

	err := conn.QueryRow(ctx, `
		SELECT MIN(start_lsn)::text, MAX(end_lsn)::text
		FROM pg_available_wal_summaries()
	`).Scan(&oldestStart, &newestEnd)
	if err != nil {
		return summaryWindow{}, fmt.Errorf("read WAL summary window: %w", err)
	}

	if oldestStart == nil || newestEnd == nil {
		return summaryWindow{}, nil
	}

	start, err := walmath.ParseLSN(*oldestStart)
	if err != nil {
		return summaryWindow{}, fmt.Errorf("parse oldest summary start_lsn: %w", err)
	}

	end, err := walmath.ParseLSN(*newestEnd)
	if err != nil {
		return summaryWindow{}, fmt.Errorf("parse newest summary end_lsn: %w", err)
	}

	return summaryWindow{hasAny: true, oldestStart: start, newestEnd: end}, nil
}

// measureSummarizerLag returns how far the summarizer's coverage trails
// pg_current_wal_lsn, in bytes. This is an instantaneous snapshot: the
// "catching up vs falling behind" distinction is made not from a rate but by
// re-sampling — CheckSummarizerReadiness maps the snapshot to GoIncremental /
// Wait / FullSameChain, and waitForSummarizer re-checks on each poll, so a lag
// that shrinks back under the go-threshold within the window proceeds and one
// that stays high collapses to a FULL.
func measureSummarizerLag(ctx context.Context, conn *pgx.Conn) (int64, error) {
	var lagBytes int64

	err := conn.QueryRow(ctx, `
		SELECT COALESCE(pg_wal_lsn_diff(
			pg_current_wal_lsn(),
			(SELECT MAX(end_lsn) FROM pg_available_wal_summaries())
		), 0)::bigint
	`).Scan(&lagBytes)
	if err != nil {
		return 0, fmt.Errorf("measure summarizer lag: %w", err)
	}

	if lagBytes < 0 {
		return 0, nil
	}

	return lagBytes, nil
}

func waitWindowForCadence(cadence time.Duration) time.Duration {
	quarter := cadence / 4
	if quarter > summarizerWaitCap {
		return summarizerWaitCap
	}

	return quarter
}

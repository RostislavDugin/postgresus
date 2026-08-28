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
// The caller maps each value to one of: run the INCR, poll and recheck, abandon
// this attempt so the INCR cadence brings the next one, or spawn a FULL
// anchoring a new chain.
type SummarizerDecision int

const (
	DecisionGoIncremental SummarizerDecision = iota
	DecisionWait
	DecisionRetryNextCadence
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
	// summarizerWaitPollInterval bounds how often the bounded wait re-probes.
	// Five seconds is tight enough that recovery is felt quickly, loose enough
	// that we don't hammer the source catalog.
	summarizerWaitPollInterval = 5 * time.Second

	// summarizerWaitCap bounds the DecisionWait timeout, which is
	// min(cadence/4, this).
	summarizerWaitCap = 30 * time.Minute
)

// CheckSummarizerReadiness classifies the state of the WAL summarizer relative to
// prevStopLSN (the parent backup's stop_lsn, or — for the WAL-gap fallback
// path — the current LSN). The conn must be an ordinary, non-replication
// connection.
//
// WAL summary files close only at checkpoint records, so a summarizer that has
// read every byte still publishes nothing until the next checkpoint.
// pg_basebackup --incremental forces its own checkpoint at backup start, which
// closes a summary file at exactly the LSN it then waits for, so how far the
// published summaries trail current WAL never decides the outcome. Liveness is
// all that is worth probing here: a summarizer that absorbs no WAL record for a
// minute is caught by PostgreSQL itself, which fails the backup with
// "WAL summarization is not progressing".
func CheckSummarizerReadiness(
	ctx context.Context,
	conn *pgx.Conn,
	prevStopLSN walmath.LSN,
	incrementalCadence time.Duration,
) (SummarizerResult, error) {
	isEnabled, err := isSummarizerEnabled(ctx, conn)
	if err != nil {
		return SummarizerResult{}, err
	}

	if !isEnabled {
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

	// Summarizer on but nothing published yet: not expiry, just not ready.
	if !window.hasAny {
		return newWaitResult(incrementalCadence), nil
	}

	isRunning, err := isSummarizerRunning(ctx, conn)
	if err != nil {
		return SummarizerResult{}, err
	}

	if !isRunning {
		return newWaitResult(incrementalCadence), nil
	}

	return SummarizerResult{Decision: DecisionGoIncremental}, nil
}

func newWaitResult(incrementalCadence time.Duration) SummarizerResult {
	return SummarizerResult{
		Decision:  DecisionWait,
		WaitFor:   waitWindowForCadence(incrementalCadence),
		PollEvery: summarizerWaitPollInterval,
	}
}

// summarizerCheck is the readiness probe behind a package var so the
// bounded-wait loop can be unit-tested without a live summarizer.
var summarizerCheck = CheckSummarizerReadiness

// resolveSummarizerDecision runs the readiness probe and, when it reports
// DecisionWait (no summaries published yet, or the summarizer process is gone),
// polls until the state resolves or the bounded window elapses. The returned
// decision is never DecisionWait: a window that expires unresolved collapses to
// DecisionRetryNextCadence, so the caller fails this attempt instead of racing a
// summarizer that is not coming back within the tick.
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
// intermediate failure row.
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
				return SummarizerResult{Decision: DecisionRetryNextCadence}, nil
			}
		}
	}
}

func isSummarizerEnabled(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var setting string

	if err := conn.QueryRow(
		ctx,
		"SELECT setting FROM pg_settings WHERE name = 'summarize_wal'",
	).Scan(&setting); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, fmt.Errorf("read summarize_wal setting: %w", err)
	}

	return setting == "on", nil
}

// pg_get_wal_summarizer_state() reports a NULL pid once the summarizer has
// exited, and collapses pending_lsn onto summarized_lsn at the same moment, so
// the pid is the only field that tells a dead summarizer from an idle one.
func isSummarizerRunning(ctx context.Context, conn *pgx.Conn) (bool, error) {
	var isRunning bool

	if err := conn.QueryRow(
		ctx,
		"SELECT summarizer_pid IS NOT NULL FROM pg_get_wal_summarizer_state()",
	).Scan(&isRunning); err != nil {
		return false, fmt.Errorf("read WAL summarizer state: %w", err)
	}

	return isRunning, nil
}

type summaryWindow struct {
	// hasAny is false when the summarizer is on but has produced no summary file yet —
	// just enabled, or idle before the first checkpoint.
	hasAny      bool
	oldestStart walmath.LSN
}

func readSummaryWindow(ctx context.Context, conn *pgx.Conn) (summaryWindow, error) {
	var oldestStart *string

	err := conn.QueryRow(ctx, `
		SELECT MIN(start_lsn)::text
		FROM pg_available_wal_summaries()
	`).Scan(&oldestStart)
	if err != nil {
		return summaryWindow{}, fmt.Errorf("read WAL summary window: %w", err)
	}

	if oldestStart == nil {
		return summaryWindow{}, nil
	}

	start, err := walmath.ParseLSN(*oldestStart)
	if err != nil {
		return summaryWindow{}, fmt.Errorf("parse oldest summary start_lsn: %w", err)
	}

	return summaryWindow{hasAny: true, oldestStart: start}, nil
}

func waitWindowForCadence(cadence time.Duration) time.Duration {
	// ApproxPeriod returns 0 for cron intervals; a zero window would skip the
	// wait entirely.
	if cadence == 0 {
		return summarizerWaitCap
	}

	quarter := cadence / 4
	if quarter > summarizerWaitCap {
		return summarizerWaitCap
	}

	return quarter
}

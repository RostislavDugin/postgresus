package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

func (s *WalStreamSupervisor) rebuildSlot(ctx context.Context, logger *slog.Logger, reason walBreakReason) error {
	s.rebuildMu.Lock()
	defer s.rebuildMu.Unlock()

	if !s.recordRebuildAttemptWithinCap() {
		return fmt.Errorf(
			"rebuild loop-protection tripped: more than %d rebuilds in the last hour", slotRebuildMaxAttemptsPerHour,
		)
	}

	logger.Info("starting slot rebuild", "reason", string(reason))

	s.isPaused.Store(true)
	defer s.isPaused.Store(false)

	s.signalRestart()

	released, err := s.waitForSlotReleased(ctx, logger)
	if err != nil {
		return fmt.Errorf("wait for our receiver to release slot: %w", err)
	}

	if !released {
		terminated, terminateErr := s.terminateOwnedSlotBackend(ctx, logger)
		if terminateErr != nil {
			return fmt.Errorf("terminate owned slot backend: %w", terminateErr)
		}

		if !terminated {
			return errors.New(
				"slot still active after stopping our receiver; refusing to drop a slot held by another consumer",
			)
		}

		released, err = s.waitForSlotReleased(ctx, logger)
		if err != nil {
			return fmt.Errorf("wait for terminated receiver to release slot: %w", err)
		}

		if !released {
			return errors.New("owned slot backend did not release the slot after termination")
		}
	}

	if err := s.spec.SourceDB.DropWalSlot(ctx, logger, s.spec.FieldEncryptor); err != nil {
		return fmt.Errorf("drop slot: %w", err)
	}

	if err := s.spec.SourceDB.VerifyWalSlot(ctx, logger, s.spec.FieldEncryptor); err != nil {
		return fmt.Errorf("recreate slot: %w", err)
	}

	if s.spec.OnSlotRebuilt != nil {
		if err := s.spec.OnSlotRebuilt(ctx, string(reason)); err != nil {
			return fmt.Errorf("request full after slot rebuild: %w", err)
		}
	}

	logger.Info("slot rebuild complete; resuming pg_receivewal on fresh slot")

	return nil
}

// isOwnedReceiverBackend reports whether an active slot is held by one of our own
// pg_receivewal processes — its PGAPPNAME carries receivewalApplicationNamePrefix.
// A slot held by anything else (a third-party consumer, or a backend we cannot
// attribute) must never be force-terminated or dropped during a rebuild.
func isOwnedReceiverBackend(state *SlotState) bool {
	return state.ActivePID != nil && strings.HasPrefix(state.ApplicationName, receivewalApplicationNamePrefix)
}

func (s *WalStreamSupervisor) terminateOwnedSlotBackend(ctx context.Context, logger *slog.Logger) (bool, error) {
	conn, err := s.spec.SourceDB.OpenInspectionConn(ctx, s.spec.FieldEncryptor)
	if err != nil {
		return false, fmt.Errorf("open inspection connection: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := InspectSlot(ctx, conn, s.slotName)
	if err != nil {
		return false, err
	}

	if state == nil || !state.Active {
		return true, nil
	}

	if !isOwnedReceiverBackend(state) {
		return false, nil
	}

	var terminated bool
	if err := conn.QueryRow(ctx, "SELECT pg_terminate_backend($1)", *state.ActivePID).Scan(&terminated); err != nil {
		return false, err
	}

	logger.Warn("terminated Databasus-owned wal receiver backend", "active_pid", *state.ActivePID)

	return terminated, nil
}

func (s *WalStreamSupervisor) waitForSlotReleased(ctx context.Context, logger *slog.Logger) (bool, error) {
	deadline := time.Now().UTC().Add(rebuildReceiverStopTimeout)

	for {
		active, ok := s.isSlotActive(ctx, logger)
		if ok && !active {
			return true, nil
		}

		if time.Now().UTC().After(deadline) {
			return ok && !active, nil
		}

		if !sleepCtx(ctx, rebuildReceiverStopPoll) {
			return false, ctx.Err()
		}
	}
}

func (s *WalStreamSupervisor) isSlotActive(ctx context.Context, logger *slog.Logger) (active, ok bool) {
	conn, err := s.spec.SourceDB.OpenInspectionConn(ctx, s.spec.FieldEncryptor)
	if err != nil {
		logger.Debug("rebuild: source unreachable while waiting for slot release", "error", err)

		return false, false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := InspectSlot(ctx, conn, s.slotName)
	if err != nil {
		return false, false
	}

	if state == nil {
		return false, true
	}

	return state.Active, true
}

func (s *WalStreamSupervisor) recordRebuildAttemptWithinCap() bool {
	now := time.Now().UTC()
	cutoff := now.Add(-time.Hour)

	kept := s.rebuildTimestamps[:0]
	for _, ts := range s.rebuildTimestamps {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}

	s.rebuildTimestamps = kept

	if len(s.rebuildTimestamps) >= slotRebuildMaxAttemptsPerHour {
		return false
	}

	s.rebuildTimestamps = append(s.rebuildTimestamps, now)

	return true
}

package usecases_physical_postgresql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/util/walmath"
)

// SlotState captures the snapshot of one physical replication slot. Sourced
// from pg_replication_slots; lag_bytes is computed against pg_current_wal_lsn().
type SlotState struct {
	SlotName        string
	Active          bool
	ActivePID       *int
	ApplicationName string
	WalStatus       string
	RestartLSN      walmath.LSN
	FlushLSN        walmath.LSN
	LagBytes        int64
}

// InspectSlot reads the row for slotName from pg_replication_slots and
// computes lag_bytes against the cluster's current WAL position. Returns nil
// when the slot does not exist (caller is responsible for the absence
// branch — typically "create slot" or "refuse" depending on context).
func InspectSlot(ctx context.Context, conn *pgx.Conn, slotName string) (*SlotState, error) {
	state := &SlotState{SlotName: slotName}

	var restartLSN, flushLSN walmath.LSN

	err := conn.QueryRow(ctx, `
		SELECT
			s.active,
			s.active_pid,
			COALESCE(r.application_name, ''),
			COALESCE(s.wal_status, ''),
			COALESCE(s.restart_lsn, '0/0'::pg_lsn),
			COALESCE(s.confirmed_flush_lsn, '0/0'::pg_lsn),
			COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), s.restart_lsn), 0)::bigint
		FROM pg_replication_slots s
		LEFT JOIN pg_stat_replication r ON r.pid = s.active_pid
		WHERE s.slot_name = $1
	`, slotName).Scan(
		&state.Active,
		&state.ActivePID,
		&state.ApplicationName,
		&state.WalStatus,
		&restartLSN,
		&flushLSN,
		&state.LagBytes,
	)

	switch {
	case err == nil:
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil

	default:
		return nil, err
	}

	state.RestartLSN = restartLSN
	state.FlushLSN = flushLSN

	return state, nil
}

package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"databasus-backend/internal/util/walmath"
)

// uploaderPollInterval — the uploader scans watch_dir this often for newly
// rotated segments. Segments rotate at the source write rate; a tight loop
// keeps local dwell time low without measurable CPU.
const uploaderPollInterval = 1 * time.Second

// runUploaderLoop scans the watch dir on a tight interval and hands each
// finalized segment / .history file to the appropriate handler.
func (s *WalStreamSupervisor) runUploaderLoop(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(uploaderPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.scanAndUpload(ctx, logger)
		}
	}
}

// recoverLocalSegmentsOnStartup sweeps watch_dir once at startup, taking over any
// finalized segment / .history left by a crash. Uses the uploader's takeover path
// so a segment whose pre-crash claim row is still file_name NULL gets finished
// rather than left for the cleaner.
func (s *WalStreamSupervisor) recoverLocalSegmentsOnStartup(ctx context.Context, logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		switch {
		case walmath.IsWalFilename(name):
			if err := s.uploader.RecoverSegment(ctx, filepath.Join(s.watchDir, name), name); err != nil {
				logger.Warn("startup wal recovery failed; live loop will retry", "wal_filename", name, "error", err)
			}

		case strings.HasSuffix(name, ".history"):
			s.handleHistoryFile(ctx, logger, name)
		}
	}
}

func (s *WalStreamSupervisor) scanAndUpload(ctx context.Context, logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		logger.Error("read wal watch dir", "error", err)

		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		switch {
		case walmath.IsWalFilename(name):
			if err := s.uploader.ProcessSegment(ctx, filepath.Join(s.watchDir, name), name); err != nil {
				logger.Warn("wal segment upload failed; will retry next tick", "wal_filename", name, "error", err)
			}

		case strings.HasSuffix(name, ".history"):
			s.handleHistoryFile(ctx, logger, name)
		}
	}
}

// handleHistoryFile uploads a .history file the receiver dropped into watch_dir
// (reusing UploadHistoryFile, which reads the body from the source cluster and is
// idempotent on (database_id, timeline_id)), then removes the local copy.
func (s *WalStreamSupervisor) handleHistoryFile(ctx context.Context, logger *slog.Logger, name string) {
	timelineID, err := parseHistoryTimeline(name)
	if err != nil {
		logger.Warn("skip unparseable history file", "name", name, "error", err)

		return
	}

	conn, err := s.spec.SourceDB.OpenInspectionConn(ctx, s.spec.FieldEncryptor)
	if err != nil {
		logger.Warn("could not open connection to upload history file; will retry", "error", err)

		return
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if _, err := UploadHistoryFile(
		ctx, conn, timelineID, s.spec.Storage, s.spec.SourceDB, s.spec.StorageID,
		s.spec.HistoryRepo, s.spec.Encryption, s.spec.MasterKey, s.spec.FieldEncryptor, logger,
	); err != nil {
		logger.Warn("history upload failed; will retry next tick", "timeline_id", timelineID, "error", err)

		return
	}

	logger.Info("timeline switch observed via .history", "timeline_id", timelineID)

	if err := os.Remove(filepath.Join(s.watchDir, name)); err != nil && !os.IsNotExist(err) {
		logger.Warn("failed to remove uploaded history file", "name", name, "error", err)
	}
}

func (s *WalStreamSupervisor) removePartials(logger *slog.Logger) {
	entries, err := os.ReadDir(s.watchDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".partial") {
			continue
		}

		if err := os.Remove(filepath.Join(s.watchDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			logger.Warn("failed to remove partial wal file", "name", entry.Name(), "error", err)
		}
	}
}

// parseHistoryTimeline extracts the timeline id from a "%08X.history" filename.
func parseHistoryTimeline(name string) (int, error) {
	trimmed := strings.TrimSuffix(name, ".history")
	if trimmed == name {
		return 0, fmt.Errorf("not a history filename: %q", name)
	}

	timelineID, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("parse timeline from %q: %w", name, err)
	}

	return int(timelineID), nil
}

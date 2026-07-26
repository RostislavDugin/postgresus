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

// Segments rotate at the source write rate; a tight loop keeps local dwell time
// low without measurable CPU.
const uploaderPollInterval = 1 * time.Second

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

// Uses the uploader's takeover path so a segment whose pre-crash claim row is
// still file_name NULL gets finished rather than left for the cleaner.
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
			s.archiveTimelineHistoryFile(ctx, logger, name)
		}
	}

	s.sweepPendingUploads(ctx, logger, s.uploader.RecoverSegment)
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
			s.archiveTimelineHistoryFile(ctx, logger, name)
		}
	}

	s.sweepPendingUploads(ctx, logger, s.uploader.ProcessSegment)
}

// Segments moved out of pg_receivewal's resume path (wal_resume.go) are valid
// WAL of the older chain, so they keep flowing to storage from here.
func (s *WalStreamSupervisor) sweepPendingUploads(
	ctx context.Context,
	logger *slog.Logger,
	upload func(ctx context.Context, localPath, walFilename string) error,
) {
	pendingUploadDir := filepath.Join(s.watchDir, pendingUploadDirName)

	entries, err := os.ReadDir(pendingUploadDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !walmath.IsWalFilename(entry.Name()) {
			continue
		}

		if err := upload(ctx, filepath.Join(pendingUploadDir, entry.Name()), entry.Name()); err != nil {
			logger.Warn("pending wal segment upload failed; will retry next tick",
				"wal_filename", entry.Name(), "error", err)
		}
	}
}

// UploadHistoryFile reads the body from the source cluster and is idempotent on
// (database_id, timeline_id), so re-processing the same file is free.
func (s *WalStreamSupervisor) archiveTimelineHistoryFile(ctx context.Context, logger *slog.Logger, name string) {
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

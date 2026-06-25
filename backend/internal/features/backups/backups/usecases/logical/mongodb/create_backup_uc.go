package usecases_logical_mongodb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	mongodbtypes "databasus-backend/internal/features/databases/databases/mongodb"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/encryption"
	io_utils "databasus-backend/internal/util/io"
	"databasus-backend/internal/util/tools"
)

const (
	backupTimeout            = 23 * time.Hour
	shutdownCheckInterval    = 1 * time.Second
	copyBufferSize           = 8 * 1024 * 1024
	progressReportIntervalMB = 1.0
)

var (
	errBackupShutdown = errors.New("backup cancelled due to shutdown")
	errBackupTimeout  = errors.New("backup cancelled due to timeout")
)

type CreateMongodbBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
	fieldEncryptor   encryption.FieldEncryptor
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func (uc *CreateMongodbBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	db *databases.Database,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	uc.logger.Info(
		"Creating MongoDB backup via mongodump",
		"databaseId", db.ID,
		"storageId", storage.ID,
	)

	mdb := db.Mongodb
	if mdb == nil {
		return nil, fmt.Errorf("mongodb database configuration is required")
	}

	if mdb.Database == "" {
		return nil, fmt.Errorf("database name is required for mongodump backups")
	}

	decryptedPassword, err := uc.fieldEncryptor.Decrypt(mdb.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database password: %w", err)
	}

	rawSizeMB, err := mdb.GetRawDbSizeMb(ctx, uc.logger, uc.fieldEncryptor)
	if err != nil {
		uc.logger.Warn("failed to fetch raw db size before backup",
			"database_id", db.ID,
			"error", err)
	} else {
		backup.BackupRawDbSizeMb = rawSizeMB
	}

	args := uc.buildMongodumpArgs(mdb, decryptedPassword)

	return uc.streamToStorage(
		ctx,
		backup,
		backupConfig,
		tools.GetMongodbExecutable(tools.MongodbExecutableMongodump),
		args,
		storage,
		backupProgressListener,
	)
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpArgs(
	mdb *mongodbtypes.MongodbDatabase,
	password string,
) []string {
	uri := mdb.BuildConnectionURI(password)

	args := []string{
		"--uri=" + uri,
		"--archive",
		"--gzip",
	}

	for _, collection := range mdb.ExcludeCollections {
		args = append(args, "--excludeCollection="+collection)
	}

	// Use numParallelCollections based on CPU count
	// Cap between 1 and 16 to balance performance and resource usage
	parallelCollections := max(1, min(mdb.CpuCount, 16))
	if parallelCollections > 1 {
		args = append(args, "--numParallelCollections="+fmt.Sprintf("%d", parallelCollections))
	}

	return args
}

func (uc *CreateMongodbBackupUsecase) streamToStorage(
	parentCtx context.Context,
	backup *backups_core_logical.LogicalBackup,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	mongodumpBin string,
	args []string,
	storage *storages.Storage,
	backupProgressListener func(completedMBs float64),
) (*backups_core_logical.BackupMetadata, error) {
	uc.logger.Info("Streaming MongoDB backup to storage", "mongodumpBin", mongodumpBin)

	ctx, cancel := uc.createBackupContext(parentCtx)
	defer cancel(nil)

	cmd := exec.CommandContext(ctx, mongodumpBin, args...)

	safeArgs := make([]string, len(args))
	for i, arg := range args {
		if len(arg) > 6 && arg[:6] == "--uri=" {
			safeArgs[i] = "--uri=mongodb://***:***@***"
		} else {
			safeArgs[i] = arg
		}
	}
	uc.logger.Info("Executing MongoDB backup command", "command", mongodumpBin, "args", safeArgs)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	pgStdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	pgStderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	stderrCh := make(chan []byte, 1)
	go func() {
		stderrOutput, _ := io.ReadAll(pgStderr)
		stderrCh <- stderrOutput
	}()

	storageReader, storageWriter := io.Pipe()

	finalWriter, encryptionWriter, backupMetadata, err := uc.setupBackupEncryption(
		backup.ID,
		backupConfig,
		storageWriter,
	)
	if err != nil {
		return nil, err
	}

	countingWriter := io_utils.NewCountingWriter(finalWriter)

	saveErrCh := make(chan error, 1)
	go func() {
		saveErr := storage.SaveFile(
			ctx,
			uc.fieldEncryptor,
			uc.logger,
			backup.FileName,
			storageReader,
		)
		if saveErr != nil {
			_ = storageReader.CloseWithError(saveErr)
			cancel(saveErr)
		}
		saveErrCh <- saveErr
	}()

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", filepath.Base(mongodumpBin), err)
	}

	copyResultCh := make(chan error, 1)
	bytesWrittenCh := make(chan int64, 1)
	go func() {
		bytesWritten, copyErr := uc.copyWithShutdownCheck(
			ctx,
			countingWriter,
			pgStdout,
			backupProgressListener,
		)
		bytesWrittenCh <- bytesWritten
		copyResultCh <- copyErr
	}()

	copyErr := <-copyResultCh
	bytesWritten := <-bytesWrittenCh
	waitErr := cmd.Wait()

	select {
	case earlySaveErr := <-saveErrCh:
		if earlySaveErr != nil {
			_ = uc.closeWriters(encryptionWriter, storageWriter)
			return nil, fmt.Errorf("save to storage: %w", earlySaveErr)
		}
		saveErrCh <- nil
	default:
	}

	select {
	case <-ctx.Done():
		uc.cleanupOnCancellation(encryptionWriter, storageWriter, saveErrCh)
		return nil, uc.classifyCancellation(ctx)
	default:
	}

	if err := uc.closeWriters(encryptionWriter, storageWriter); err != nil {
		<-saveErrCh
		return nil, err
	}

	saveErr := <-saveErrCh
	stderrOutput := <-stderrCh

	if waitErr == nil && copyErr == nil && saveErr == nil && backupProgressListener != nil {
		sizeMB := float64(bytesWritten) / (1024 * 1024)
		backupProgressListener(sizeMB)
	}

	switch {
	case waitErr != nil:
		return nil, uc.buildMongodumpErrorMessage(waitErr, stderrOutput, mongodumpBin)
	case copyErr != nil:
		return nil, fmt.Errorf("copy to storage: %w", copyErr)
	case saveErr != nil:
		return nil, fmt.Errorf("save to storage: %w", saveErr)
	}

	return &backupMetadata, nil
}

func (uc *CreateMongodbBackupUsecase) createBackupContext(
	parentCtx context.Context,
) (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(parentCtx)

	timeout := time.AfterFunc(backupTimeout, func() { cancel(errBackupTimeout) })

	go func() {
		defer timeout.Stop()

		ticker := time.NewTicker(shutdownCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if config.IsShouldShutdown() {
					cancel(errBackupShutdown)
					return
				}
			}
		}
	}()

	return ctx, cancel
}

func (uc *CreateMongodbBackupUsecase) setupBackupEncryption(
	backupID uuid.UUID,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	storageWriter io.WriteCloser,
) (io.Writer, *backup_encryption.EncryptionWriter, backups_core_logical.BackupMetadata, error) {
	backupMetadata := backups_core_logical.BackupMetadata{
		BackupID:   backupID,
		Encryption: backups_core_enums.BackupEncryptionNone,
	}

	if backupConfig.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		return storageWriter, nil, backupMetadata, nil
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, nil, backupMetadata, fmt.Errorf("failed to get master key: %w", err)
	}

	encSetup, err := backup_encryption.SetupEncryptionWriter(storageWriter, masterKey, backupID)
	if err != nil {
		return nil, nil, backupMetadata, err
	}

	backupMetadata.Encryption = backups_core_enums.BackupEncryptionEncrypted
	backupMetadata.EncryptionSalt = &encSetup.SaltBase64
	backupMetadata.EncryptionIV = &encSetup.NonceBase64

	return encSetup.Writer, encSetup.Writer, backupMetadata, nil
}

func (uc *CreateMongodbBackupUsecase) copyWithShutdownCheck(
	ctx context.Context,
	dst io.Writer,
	src io.Reader,
	backupProgressListener func(completedMBs float64),
) (int64, error) {
	buf := make([]byte, copyBufferSize)
	var totalWritten int64
	var lastReportedMB float64

	for {
		select {
		case <-ctx.Done():
			return totalWritten, ctx.Err()
		default:
		}

		if config.IsShouldShutdown() {
			return totalWritten, errors.New("shutdown requested")
		}

		nr, readErr := src.Read(buf)
		if nr > 0 {
			writeResultCh := make(chan writeResult, 1)
			go func() {
				nw, writeErr := dst.Write(buf[:nr])
				writeResultCh <- writeResult{nw, writeErr}
			}()

			var nw int
			var writeErr error

			select {
			case <-ctx.Done():
				return totalWritten, fmt.Errorf("copy cancelled during write: %w", ctx.Err())
			case result := <-writeResultCh:
				nw = result.bytesWritten
				writeErr = result.writeErr
			}

			if nw < 0 || nr < nw {
				nw = 0
				if writeErr == nil {
					writeErr = fmt.Errorf("invalid write result")
				}
			}

			if writeErr != nil {
				return totalWritten, writeErr
			}
			if nr != nw {
				return totalWritten, io.ErrShortWrite
			}
			totalWritten += int64(nw)

			if backupProgressListener != nil {
				currentMB := float64(totalWritten) / (1024 * 1024)
				if currentMB-lastReportedMB >= progressReportIntervalMB {
					backupProgressListener(currentMB)
					lastReportedMB = currentMB
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return totalWritten, nil
			}
			return totalWritten, readErr
		}
	}
}

func (uc *CreateMongodbBackupUsecase) cleanupOnCancellation(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter *io.PipeWriter,
	saveErrCh chan error,
) {
	if encryptionWriter != nil {
		_ = encryptionWriter.Close()
	}
	_ = storageWriter.CloseWithError(errors.New("backup cancelled"))
	<-saveErrCh
}

func (uc *CreateMongodbBackupUsecase) closeWriters(
	encryptionWriter *backup_encryption.EncryptionWriter,
	storageWriter *io.PipeWriter,
) error {
	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			uc.logger.Error("Failed to close encryption writer", "error", err)
			return fmt.Errorf("failed to close encryption writer: %w", err)
		}
	}
	if err := storageWriter.Close(); err != nil {
		uc.logger.Error("Failed to close storage writer", "error", err)
		return fmt.Errorf("failed to close storage writer: %w", err)
	}
	return nil
}

func (uc *CreateMongodbBackupUsecase) classifyCancellation(ctx context.Context) error {
	cause := context.Cause(ctx)

	switch {
	case errors.Is(cause, errBackupShutdown):
		return errors.New("backup cancelled due to shutdown")
	case errors.Is(cause, errBackupTimeout):
		return errors.New("backup cancelled due to timeout")
	case cause == nil, errors.Is(cause, context.Canceled), errors.Is(cause, context.DeadlineExceeded):
		return errors.New("backup cancelled")
	default:
		return fmt.Errorf("save to storage: %w", cause)
	}
}

func (uc *CreateMongodbBackupUsecase) buildMongodumpErrorMessage(
	waitErr error,
	stderrOutput []byte,
	mongodumpBin string,
) error {
	stderrStr := string(stderrOutput)

	if len(stderrStr) > 0 {
		return fmt.Errorf(
			"%s failed: %w\nstderr: %s",
			filepath.Base(mongodumpBin),
			waitErr,
			stderrStr,
		)
	}

	return fmt.Errorf("%s failed: %w", filepath.Base(mongodumpBin), waitErr)
}

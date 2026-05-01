package usecases_clickhouse

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"databasus-backend/internal/config"
	chmanifest "databasus-backend/internal/features/backups/backups/clickhouse_manifest"
	backups_core "databasus-backend/internal/features/backups/backups/core"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	backups_config "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/features/databases"
	chtypes "databasus-backend/internal/features/databases/databases/clickhouse"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/storages"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

const (
	restoreTimeout        = 23 * time.Hour
	shutdownCheckInterval = 1 * time.Second
)

type RestoreClickhouseBackupUsecase struct {
	logger           *slog.Logger
	secretKeyService *encryption_secrets.SecretKeyService
}

// pendingTable accumulates per-table state across the tar walk. The DDL
// arrives in restore.sql, then the meta sidecar, then the native bytes
// (which we spool to disk). After the walk completes and footer + cross-
// checks pass, the apply phase reads spoolPath into INSERT FORMAT Native.
type pendingTable struct {
	tableID         string
	tableName       string
	originalDDL     string
	observedDDLHash string
	meta            *chmanifest.NativeMeta
	spoolPath       string
	inserted        bool
}

type mvEntry struct {
	id              string
	originalDDL     string
	observedDDLHash string
}

// restoreState is built up by the tar walk and consumed by the apply phase.
// Splitting these two phases gives the restore atomic-or-rollback semantics
// at the granularity of the entire archive: target is untouched until every
// byte and every checksum has been verified.
type restoreState struct {
	header        chmanifest.Header
	footer        chmanifest.Footer
	headerLoaded  bool
	footerLoaded  bool
	pending       map[string]*pendingTable
	mvDDLPending  []mvEntry
	mvAppliedSet  map[string]bool
	seenTarNames  map[string]bool
}

func (uc *RestoreClickhouseBackupUsecase) Execute(
	parentCtx context.Context,
	originalDB *databases.Database,
	restoringToDB *databases.Database,
	_ *backups_config.BackupConfig,
	restore restores_core.Restore,
	backup *backups_core.Backup,
	storage *storages.Storage,
) error {
	if originalDB.Type != databases.DatabaseTypeClickhouse {
		return errors.New("database type not supported")
	}

	uc.logger.Info(
		"restoring ClickHouse backup",
		"restoreId", restore.ID,
		"backupId", backup.ID,
	)

	ch := restoringToDB.Clickhouse
	if ch == nil {
		return errors.New("clickhouse configuration is required for restore")
	}
	if ch.Database == "" {
		return errors.New("target database name is required")
	}

	fieldEncryptor := util_encryption.GetFieldEncryptor()
	password, err := fieldEncryptor.Decrypt(restoringToDB.ID, ch.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	ctx, cancel := uc.createRestoreContext(parentCtx)
	defer cancel()

	rawReader, err := storage.GetFile(fieldEncryptor, backup.FileName)
	if err != nil {
		return fmt.Errorf("failed to open backup file from storage: %w", err)
	}
	defer func() { _ = rawReader.Close() }()

	decReader, err := uc.setupDecryption(rawReader, backup)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(os.TempDir(), "ch-restore-"+backup.ID.String()+"-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	if err := os.Chmod(tempDir, 0o700); err != nil {
		_ = os.RemoveAll(tempDir)
		return fmt.Errorf("chmod temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	cliBin := tools.GetClickhouseExecutable(
		tools.ClickhouseExecutableClient,
		config.GetEnv().EnvMode,
		config.GetEnv().ClickhouseInstallDir,
	)
	if _, err := exec.LookPath(cliBin); err != nil {
		return fmt.Errorf("clickhouse-client binary not accessible at %s: %w", cliBin, err)
	}

	state := &restoreState{
		pending:      map[string]*pendingTable{},
		mvAppliedSet: map[string]bool{},
		seenTarNames: map[string]bool{},
	}

	if err := uc.validatePhase(ctx, decReader, ch, tempDir, state); err != nil {
		return err
	}

	if err := uc.applyPhase(ctx, ch, password, cliBin, state); err != nil {
		return err
	}

	uc.logger.Info(
		"clickhouse restore completed",
		"restoreId", restore.ID,
		"tableCount", len(state.footer.Tables),
		"mvCount", len(state.mvAppliedSet),
	)
	return nil
}

// validatePhase walks the tar archive, spools native payloads to disk,
// verifies every meta sidecar against its native bytes, parses the footer,
// and runs the four set-equality cross-checks. The target ClickHouse
// instance is not touched in this phase — any error here aborts the restore
// with the target unchanged.
func (uc *RestoreClickhouseBackupUsecase) validatePhase(
	ctx context.Context,
	decReader io.Reader,
	ch *chtypes.ClickhouseDatabase,
	tempDir string,
	state *restoreState,
) error {
	tr := tar.NewReader(decReader)

	for {
		h, readErr := tr.Next()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return fmt.Errorf("tar read: %w", readErr)
		}

		if config.IsShouldShutdown() {
			return errors.New("restore cancelled due to shutdown")
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := rejectNonCanonicalPath(h.Name); err != nil {
			return err
		}
		if state.seenTarNames[h.Name] {
			return fmt.Errorf("duplicate tar entry %q (malformed archive)", h.Name)
		}
		state.seenTarNames[h.Name] = true

		if err := uc.validateEntry(tr, h, ch, tempDir, state); err != nil {
			return err
		}
	}

	if !state.headerLoaded {
		return errors.New("manifest.header.json missing from backup archive")
	}
	if !state.footerLoaded {
		return errors.New("manifest.footer.json missing from backup archive (likely truncated)")
	}

	return uc.crossCheckArchive(state)
}

// applyPhase mutates the target only after validatePhase has approved the
// archive. Order is: prepare target db, drop+create each table, INSERT
// FORMAT Native from each spool file, then materialized view DDL.
func (uc *RestoreClickhouseBackupUsecase) applyPhase(
	ctx context.Context,
	ch *chtypes.ClickhouseDatabase,
	password, cliBin string,
	state *restoreState,
) error {
	if err := uc.prepareTarget(ctx, ch, password, cliBin); err != nil {
		return err
	}

	for _, t := range state.header.Tables {
		p := state.pending[t.ID]
		if p == nil {
			return fmt.Errorf("internal: pending state missing for validated table %s", t.ID)
		}
		if err := uc.applyTable(ctx, ch, password, cliBin, p, state.header.SourceDatabase); err != nil {
			return err
		}
	}

	for _, mv := range state.mvDDLPending {
		mvSQL, err := deriveRestoreDDL(mv.originalDDL, state.header.SourceDatabase, ch.Database, ch.IsKeepReplicatedDDL)
		if err != nil {
			return fmt.Errorf("derive restore DDL for mv %s: %w", mv.id, err)
		}
		if err := execClient(ctx, cliBin, ch, password, mvSQL, nil); err != nil {
			return fmt.Errorf("create materialized view %s: %w", mv.id, err)
		}
		state.mvAppliedSet[mv.id] = true
	}

	if err := uc.checkMVApplyComplete(state); err != nil {
		return err
	}

	return nil
}

func (uc *RestoreClickhouseBackupUsecase) applyTable(
	ctx context.Context,
	ch *chtypes.ClickhouseDatabase,
	password, cliBin string,
	p *pendingTable,
	sourceDatabase string,
) error {
	createSQL, err := deriveRestoreDDL(p.originalDDL, sourceDatabase, ch.Database, ch.IsKeepReplicatedDDL)
	if err != nil {
		return fmt.Errorf("derive restore DDL for %s: %w", p.tableName, err)
	}

	if ch.IsDropExisting {
		dropSQL := fmt.Sprintf(
			"DROP TABLE IF EXISTS %s.%s",
			quoteIdent(ch.Database), quoteIdent(p.tableName),
		)
		if err := execClient(ctx, cliBin, ch, password, dropSQL, nil); err != nil {
			return fmt.Errorf("drop existing table %s: %w", p.tableName, err)
		}
	}

	if err := execClient(ctx, cliBin, ch, password, createSQL, nil); err != nil {
		return fmt.Errorf("create table %s: %w", p.tableName, err)
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) SETTINGS insert_allow_materialized_columns=0 FORMAT Native",
		quoteIdent(ch.Database), quoteIdent(p.tableName),
		strings.Join(quoteIdents(p.meta.Columns), ", "),
	)

	f, err := os.Open(p.spoolPath)
	if err != nil {
		return fmt.Errorf("open spool file for %s: %w", p.tableName, err)
	}

	insertErr := execClient(ctx, cliBin, ch, password, insertSQL, f)
	closeErr := f.Close()
	if insertErr != nil {
		return fmt.Errorf("insert into %s: %w", p.tableName, insertErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close spool file for %s: %w", p.tableName, closeErr)
	}

	p.inserted = true
	return nil
}

// validateEntry processes one tar entry during the read-only validation
// phase. Native payloads are spooled to disk; everything else is parsed or
// drained.
func (uc *RestoreClickhouseBackupUsecase) validateEntry(
	tr *tar.Reader,
	h *tar.Header,
	ch *chtypes.ClickhouseDatabase,
	tempDir string,
	state *restoreState,
) error {
	switch {
	case h.Name == "manifest.header.json":
		return loadHeader(tr, &state.header, &state.headerLoaded)

	case h.Name == "manifest.footer.json":
		return loadFooter(tr, &state.footer, &state.footerLoaded)

	case h.Name == "db.sql":
		// Source CREATE DATABASE — informational; target uses ENGINE = Atomic
		// regardless. Drain to advance the reader.
		_, err := io.Copy(io.Discard, tr)
		return err

	case strings.HasPrefix(h.Name, "tables/") && strings.HasSuffix(h.Name, "/restore.sql"):
		return loadTableRestoreSQL(tr, h, state)

	case strings.HasPrefix(h.Name, "tables/") && strings.HasSuffix(h.Name, "/native.meta.json"):
		return loadTableNativeMeta(tr, h, state.pending)

	case strings.HasPrefix(h.Name, "tables/") && strings.HasSuffix(h.Name, "/native"):
		return spoolAndVerifyNative(tr, h, tempDir, state.pending)

	case strings.HasPrefix(h.Name, "tables/") && strings.HasSuffix(h.Name, "/ddl.sql"):
		return loadTableOriginalDDL(tr, h, state)

	case strings.HasPrefix(h.Name, "mvs/") && strings.HasSuffix(h.Name, "/restore.sql"):
		return loadMVRestoreSQL(tr, h, state)

	case strings.HasPrefix(h.Name, "mvs/") && strings.HasSuffix(h.Name, "/ddl.sql"):
		return loadMVOriginalDDL(tr, h, state)

	default:
		return fmt.Errorf("unexpected tar entry %q", h.Name)
	}
}

func loadHeader(tr *tar.Reader, header *chmanifest.Header, loaded *bool) error {
	buf, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("read manifest.header.json: %w", err)
	}
	if err := json.Unmarshal(buf, header); err != nil {
		return fmt.Errorf("parse manifest.header.json: %w", err)
	}
	if header.SchemaVersion != chmanifest.SchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %d (expected %d)",
			header.SchemaVersion, chmanifest.SchemaVersion)
	}
	*loaded = true
	return nil
}

func loadFooter(tr *tar.Reader, footer *chmanifest.Footer, loaded *bool) error {
	buf, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("read manifest.footer.json: %w", err)
	}
	if err := json.Unmarshal(buf, footer); err != nil {
		return fmt.Errorf("parse manifest.footer.json: %w", err)
	}
	*loaded = true
	return nil
}

// loadTableRestoreSQL drains the diagnostic-only restore.sql payload. The
// canonical source for replay is ddl.sql (which is hash-verified against
// meta.DDLHash); the cleaned + engine-rewritten + identifier-rewritten
// statement is regenerated from those bytes at apply time. Trusting an
// unhashed restore.sql would let a tampered archive inject DDL that
// passed validation.
func loadTableRestoreSQL(
	tr *tar.Reader, h *tar.Header,
	state *restoreState,
) error {
	tableID, err := tarIDFromPath(h.Name, "tables/", "/restore.sql")
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, tr); err != nil {
		return fmt.Errorf("drain %s: %w", h.Name, err)
	}
	if getOrInitPending(state.pending, tableID, state.header) == nil {
		return fmt.Errorf("restore.sql arrived for table id %q not in manifest header", tableID)
	}
	return nil
}

func loadTableNativeMeta(
	tr *tar.Reader, h *tar.Header,
	pending map[string]*pendingTable,
) error {
	tableID, err := tarIDFromPath(h.Name, "tables/", "/native.meta.json")
	if err != nil {
		return err
	}
	buf, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("read %s: %w", h.Name, err)
	}
	var meta chmanifest.NativeMeta
	if err := json.Unmarshal(buf, &meta); err != nil {
		return fmt.Errorf("parse %s: %w", h.Name, err)
	}
	p, ok := pending[tableID]
	if !ok {
		return fmt.Errorf("native.meta.json arrived for unknown table id %s", tableID)
	}
	p.meta = &meta
	return nil
}

// loadTableOriginalDDL captures both the bytes and sha256 of ddl.sql. The
// bytes are the trusted source for restore-time DDL — restore.sql is
// diagnostic-only because it is unhashed in the manifest, so we regenerate
// the cleaned/rewritten DDL from ddl.sql at apply time. ddl.sql is the
// first per-table tar entry, so this is where pending state is bootstrapped
// from the header.
func loadTableOriginalDDL(
	tr *tar.Reader, h *tar.Header,
	state *restoreState,
) error {
	tableID, err := tarIDFromPath(h.Name, "tables/", "/ddl.sql")
	if err != nil {
		return err
	}
	buf, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("read %s: %w", h.Name, err)
	}
	p := getOrInitPending(state.pending, tableID, state.header)
	if p == nil {
		return fmt.Errorf("ddl.sql arrived for table id %q not in manifest header", tableID)
	}
	sum := sha256.Sum256(buf)
	p.originalDDL = string(buf)
	p.observedDDLHash = hex.EncodeToString(sum[:])
	return nil
}

// spoolAndVerifyNative copies the native payload to disk, computes its
// sha256 + size on the fly, and rejects any mismatch against the meta
// sidecar that arrived just before. The spool path is held on the pending
// entry for the apply phase to feed into INSERT FORMAT Native.
func spoolAndVerifyNative(
	tr *tar.Reader, h *tar.Header,
	tempDir string,
	pending map[string]*pendingTable,
) error {
	tableID, err := tarIDFromPath(h.Name, "tables/", "/native")
	if err != nil {
		return err
	}
	p, ok := pending[tableID]
	if !ok {
		return fmt.Errorf("native arrived for unknown table id %s", tableID)
	}
	if p.meta == nil {
		return fmt.Errorf("native arrived before native.meta.json for table id %s", tableID)
	}
	if p.originalDDL == "" {
		return fmt.Errorf("native arrived before ddl.sql for table id %s", tableID)
	}

	spoolPath, observedSize, observedSHA, err := spoolAndHash(tr, h.Size, tempDir, tableID)
	if err != nil {
		return fmt.Errorf("spool native for %s: %w", p.tableName, err)
	}

	if observedSize != p.meta.NativeBytes {
		_ = os.Remove(spoolPath)
		return fmt.Errorf(
			"native size mismatch for %s: meta=%d observed=%d",
			p.tableName, p.meta.NativeBytes, observedSize,
		)
	}
	if observedSHA != p.meta.SHA256 {
		_ = os.Remove(spoolPath)
		return fmt.Errorf(
			"native sha256 mismatch for %s: meta=%s observed=%s",
			p.tableName, p.meta.SHA256, observedSHA,
		)
	}

	p.spoolPath = spoolPath
	return nil
}

// loadMVOriginalDDL captures the verbatim mvs/<id>/ddl.sql payload. Like
// the table flow, this is the trusted source for replay; restore.sql is
// drained as diagnostic-only.
func loadMVOriginalDDL(
	tr *tar.Reader, h *tar.Header,
	state *restoreState,
) error {
	mvID, err := tarIDFromPath(h.Name, "mvs/", "/ddl.sql")
	if err != nil {
		return err
	}
	if !mvInHeader(state.header, mvID) {
		if _, err := io.Copy(io.Discard, tr); err != nil {
			return fmt.Errorf("drain unknown mv ddl %q: %w", mvID, err)
		}
		return fmt.Errorf("mv ddl.sql arrived for id %q not in manifest header", mvID)
	}
	buf, err := io.ReadAll(tr)
	if err != nil {
		return fmt.Errorf("read mv %s ddl: %w", mvID, err)
	}
	sum := sha256.Sum256(buf)
	state.mvDDLPending = append(state.mvDDLPending, mvEntry{
		id:              mvID,
		originalDDL:     string(buf),
		observedDDLHash: hex.EncodeToString(sum[:]),
	})
	return nil
}

// loadMVRestoreSQL drains the diagnostic restore.sql for the MV. The
// rewritten DDL is regenerated at apply time from the trusted ddl.sql
// captured by loadMVOriginalDDL.
func loadMVRestoreSQL(
	tr *tar.Reader, h *tar.Header,
	state *restoreState,
) error {
	mvID, err := tarIDFromPath(h.Name, "mvs/", "/restore.sql")
	if err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, tr); err != nil {
		return fmt.Errorf("drain %s: %w", h.Name, err)
	}
	if !mvInHeader(state.header, mvID) {
		return fmt.Errorf("mv restore.sql arrived for id %q not in manifest header", mvID)
	}
	return nil
}

func mvInHeader(header chmanifest.Header, mvID string) bool {
	for _, mv := range header.MaterializedViews {
		if mv.ID == mvID {
			return true
		}
	}
	return false
}

// crossCheckArchive runs four set-equality and per-table value checks that
// catch malformed-but-OK-looking archives. Tables: header == meta == footer.
// Materialized views: header == loaded-DDL list. (Inserted-table set
// equality is checked at the end of applyPhase.)
func (uc *RestoreClickhouseBackupUsecase) crossCheckArchive(state *restoreState) error {
	if state.footer.Status != "OK" {
		return fmt.Errorf("backup was not completed cleanly (footer status=%q)", state.footer.Status)
	}

	plannedDataIDs := make(map[string]bool, len(state.header.Tables))
	for _, t := range state.header.Tables {
		if plannedDataIDs[t.ID] {
			return fmt.Errorf("duplicate table id %q in manifest header", t.ID)
		}
		plannedDataIDs[t.ID] = true
	}

	metaIDs := make(map[string]bool, len(state.pending))
	for id, p := range state.pending {
		if p.meta != nil && p.spoolPath != "" {
			metaIDs[id] = true
		}
	}

	footerIDs := make(map[string]bool, len(state.footer.Tables))
	for _, ft := range state.footer.Tables {
		footerIDs[ft.ID] = true
	}

	if !setsEqual(plannedDataIDs, metaIDs) {
		return fmt.Errorf("data table ID set mismatch: header=%v vs meta=%v",
			sortedKeys(plannedDataIDs), sortedKeys(metaIDs))
	}
	if !setsEqual(metaIDs, footerIDs) {
		return fmt.Errorf("data table ID set mismatch: meta=%v vs footer=%v",
			sortedKeys(metaIDs), sortedKeys(footerIDs))
	}

	plannedMVIDs := make(map[string]bool, len(state.header.MaterializedViews))
	for _, mv := range state.header.MaterializedViews {
		if plannedMVIDs[mv.ID] {
			return fmt.Errorf("duplicate materialized view id %q in manifest header", mv.ID)
		}
		plannedMVIDs[mv.ID] = true
	}
	loadedMVIDs := make(map[string]bool, len(state.mvDDLPending))
	for _, mv := range state.mvDDLPending {
		loadedMVIDs[mv.id] = true
	}
	if !setsEqual(plannedMVIDs, loadedMVIDs) {
		return fmt.Errorf("MV ID set mismatch: header=%v vs loaded=%v",
			sortedKeys(plannedMVIDs), sortedKeys(loadedMVIDs))
	}

	for _, ft := range state.footer.Tables {
		p := state.pending[ft.ID]
		if p == nil || p.meta == nil {
			return fmt.Errorf("footer references missing table %s (id=%s)", ft.Name, ft.ID)
		}
		if p.meta.NativeBytes != ft.NativeBytes {
			return fmt.Errorf(
				"footer/meta size mismatch for %s: meta=%d footer=%d",
				ft.Name, p.meta.NativeBytes, ft.NativeBytes,
			)
		}
		if p.meta.SHA256 != ft.SHA256 {
			return fmt.Errorf(
				"footer/meta sha256 mismatch for %s: meta=%s footer=%s",
				ft.Name, p.meta.SHA256, ft.SHA256,
			)
		}
		if p.observedDDLHash == "" {
			return fmt.Errorf("ddl.sql missing for table %s (id=%s)", ft.Name, ft.ID)
		}
		if p.meta.DDLHash == "" {
			return fmt.Errorf("meta.DDLHash missing for table %s (id=%s); cannot verify ddl.sql integrity", ft.Name, ft.ID)
		}
		if p.observedDDLHash != p.meta.DDLHash {
			return fmt.Errorf(
				"ddl.sql sha256 mismatch for %s: meta=%s observed=%s",
				ft.Name, p.meta.DDLHash, p.observedDDLHash,
			)
		}
	}

	for _, mv := range state.header.MaterializedViews {
		var observed string
		var loaded bool
		for _, m := range state.mvDDLPending {
			if m.id == mv.ID {
				observed = m.observedDDLHash
				loaded = true
				break
			}
		}
		if !loaded {
			return fmt.Errorf("ddl.sql missing for materialized view %s (id=%s)", mv.Name, mv.ID)
		}
		if mv.DDLHash == "" {
			return fmt.Errorf("header.DDLHash missing for materialized view %s (id=%s); cannot verify ddl.sql integrity", mv.Name, mv.ID)
		}
		if observed != mv.DDLHash {
			return fmt.Errorf(
				"ddl.sql sha256 mismatch for materialized view %s: header=%s observed=%s",
				mv.Name, mv.DDLHash, observed,
			)
		}
	}

	return nil
}

// checkMVApplyComplete is the apply-phase counterpart to crossCheckArchive.
// It catches any silent mismatch between MVs that loaded successfully in
// validation and MVs that the apply loop actually replayed.
func (uc *RestoreClickhouseBackupUsecase) checkMVApplyComplete(state *restoreState) error {
	plannedMVIDs := make(map[string]bool, len(state.header.MaterializedViews))
	for _, mv := range state.header.MaterializedViews {
		plannedMVIDs[mv.ID] = true
	}
	if !setsEqual(plannedMVIDs, state.mvAppliedSet) {
		return fmt.Errorf("MV apply mismatch: header=%v vs applied=%v",
			sortedKeys(plannedMVIDs), sortedKeys(state.mvAppliedSet))
	}
	return nil
}

// prepareTarget creates the target database with ENGINE = Atomic. With
// IsDropExisting=true any pre-existing target is dropped first; otherwise
// a non-empty target aborts the restore.
func (uc *RestoreClickhouseBackupUsecase) prepareTarget(
	ctx context.Context,
	ch *chtypes.ClickhouseDatabase,
	password, cliBin string,
) error {
	targetDb := quoteIdent(ch.Database)

	if ch.IsDropExisting {
		if err := execClient(ctx, cliBin, ch, password,
			fmt.Sprintf("DROP DATABASE IF EXISTS %s", targetDb), nil); err != nil {
			return fmt.Errorf("drop target db: %w", err)
		}
	} else {
		if err := uc.failIfTargetHasTables(ctx, ch, password); err != nil {
			return err
		}
	}

	if err := execClient(ctx, cliBin, ch, password,
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s ENGINE = Atomic", targetDb),
		nil,
	); err != nil {
		return fmt.Errorf("create target db: %w", err)
	}

	return nil
}

// failIfTargetHasTables probes the always-present `system` database so the
// existence check works whether or not the target db has been created yet.
// All connection / query errors are propagated; only an absent target db
// counts as "empty".
func (uc *RestoreClickhouseBackupUsecase) failIfTargetHasTables(
	ctx context.Context,
	ch *chtypes.ClickhouseDatabase,
	password string,
) error {
	probeCfg := *ch
	probeCfg.Database = "system"

	conn, err := chtypes.OpenConn(ctx, &probeCfg, password)
	if err != nil {
		return fmt.Errorf("connect to target server for emptiness probe: %w", err)
	}
	defer func() { _ = conn.Close() }()

	var dbExists uint64
	if err := conn.QueryRow(ctx,
		"SELECT count() FROM system.databases WHERE name = ?", ch.Database,
	).Scan(&dbExists); err != nil {
		return fmt.Errorf("check target database existence: %w", err)
	}
	if dbExists == 0 {
		return nil
	}

	var tableCount uint64
	if err := conn.QueryRow(ctx,
		"SELECT count() FROM system.tables WHERE database = ?", ch.Database,
	).Scan(&tableCount); err != nil {
		return fmt.Errorf("count tables in target database: %w", err)
	}

	if tableCount > 0 {
		return fmt.Errorf(
			"target database %q already contains %d table(s); "+
				"set IsDropExisting=true to drop them, or restore into a fresh database",
			ch.Database, tableCount,
		)
	}
	return nil
}

// setupDecryption wraps an encrypted backup stream with the AES-GCM reader
// the rest of the engines use. Plain backups pass through.
func (uc *RestoreClickhouseBackupUsecase) setupDecryption(
	reader io.Reader,
	backup *backups_core.Backup,
) (io.Reader, error) {
	if backup.Encryption != backups_config.BackupEncryptionEncrypted {
		return reader, nil
	}
	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		return nil, errors.New("encrypted backup missing salt or IV")
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		return nil, fmt.Errorf("decode encryption salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		return nil, fmt.Errorf("decode encryption IV: %w", err)
	}

	masterKey, err := uc.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("get master key: %w", err)
	}

	decryptReader, err := backup_encryption.NewDecryptionReader(reader, masterKey, backup.ID, salt, nonce)
	if err != nil {
		return nil, fmt.Errorf("create decryption reader: %w", err)
	}
	return decryptReader, nil
}

// createRestoreContext bounds the entire restore at restoreTimeout and
// cancels early on shutdown.
func (uc *RestoreClickhouseBackupUsecase) createRestoreContext(
	parentCtx context.Context,
) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parentCtx, restoreTimeout)

	go func() {
		ticker := time.NewTicker(shutdownCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-parentCtx.Done():
				cancel()
				return
			case <-ticker.C:
				if config.IsShouldShutdown() {
					cancel()
					return
				}
			}
		}
	}()

	return ctx, cancel
}

// --- Helpers ---

// spoolAndHash copies up to size bytes from r into a 0600-perm tempfile in
// dir, returning the path, observed byte count, and sha256 hex digest.
// Caller is responsible for os.Remove on the returned path.
func spoolAndHash(r io.Reader, size int64, dir, idHint string) (string, int64, string, error) {
	tempPath := filepath.Join(dir, idHint+".native")
	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, "", err
	}
	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)
	written, copyErr := io.Copy(mw, io.LimitReader(r, size))
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(tempPath)
		return "", 0, "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return "", 0, "", closeErr
	}
	return tempPath, written, hex.EncodeToString(hasher.Sum(nil)), nil
}

// execClient runs clickhouse-client. DDL statements pass via --query with a
// nil stdin; INSERT FORMAT Native passes the prologue via --query and the
// binary payload via stdin.
func execClient(
	ctx context.Context,
	cliBin string,
	ch *chtypes.ClickhouseDatabase,
	password, query string,
	stdin io.Reader,
) error {
	args := []string{
		"--host=" + ch.Host,
		"--port=" + fmt.Sprintf("%d", ch.Port),
		"--user=" + ch.Username,
		"--query=" + query,
	}
	if ch.IsHttps {
		args = append(args, "--secure")
		if !ch.IsStrictTls {
			args = append(args, "--accept-invalid-certificate")
		}
	}

	cmd := exec.CommandContext(ctx, cliBin, args...)
	cmd.Env = filterAndAppendEnv(os.Environ(), map[string]string{
		"CLICKHOUSE_PASSWORD": password,
		"LC_ALL":              "C.UTF-8",
		"LANG":                "C.UTF-8",
	})
	if stdin != nil {
		cmd.Stdin = stdin
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clickhouse-client failed: %w (stderr: %s)",
			err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// filterAndAppendEnv builds a child env that drops any inherited
// CLICKHOUSE_* and appends the values we want. An operator-set
// CLICKHOUSE_HOST or CLICKHOUSE_PASSWORD must not silently override our
// argv flags.
func filterAndAppendEnv(parent []string, ours map[string]string) []string {
	out := make([]string, 0, len(parent)+len(ours))
	for _, e := range parent {
		if !strings.HasPrefix(e, "CLICKHOUSE_") {
			out = append(out, e)
		}
	}
	for k, v := range ours {
		out = append(out, k+"="+v)
	}
	return out
}

// deriveRestoreDDL is the deterministic transform chain we apply to the
// trusted ddl.sql payload to produce the statement we actually replay.
// Running this at apply time (rather than trusting the archive's restore.sql)
// means archive tampering of restore.sql cannot inject DDL into the target.
func deriveRestoreDDL(originalDDL, sourceDb, targetDb string, keepReplicated bool) (string, error) {
	cleaned, _, err := chtypes.CleanDDLForBackup(originalDDL)
	if err != nil {
		return "", fmt.Errorf("clean DDL: %w", err)
	}
	engineRewritten, _, err := chtypes.RewriteEngineForRestore(cleaned, keepReplicated)
	if err != nil {
		return "", fmt.Errorf("engine rewrite: %w", err)
	}
	final, err := chtypes.RewriteIdentifiersForRestore(engineRewritten, sourceDb, targetDb)
	if err != nil {
		return "", fmt.Errorf("identifier rewrite: %w", err)
	}
	return final, nil
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteIdents(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = quoteIdent(n)
	}
	return out
}

// tarIDFromPath extracts the opaque id from a strict three-segment path of
// the form `<expectedPrefix>/<id>/<expectedFile>`. Anything else returns an
// error — including canonical-but-nested forms like
// `mvs/foo/mv-abc/ddl.sql`, which would otherwise collapse to the same id
// as `mvs/mv-abc/ddl.sql` via path.Base and let an unhashed payload bypass
// validation.
func tarIDFromPath(name, expectedPrefix, expectedFile string) (string, error) {
	segs := strings.Split(name, "/")
	if len(segs) != 3 {
		return "", fmt.Errorf("malformed tar entry %q: expected three path segments", name)
	}
	if segs[0]+"/" != expectedPrefix {
		return "", fmt.Errorf("malformed tar entry %q: expected prefix %q", name, expectedPrefix)
	}
	if "/"+segs[2] != expectedFile {
		return "", fmt.Errorf("malformed tar entry %q: expected file %q", name, expectedFile)
	}
	if segs[1] == "" {
		return "", fmt.Errorf("malformed tar entry %q: empty id segment", name)
	}
	return segs[1], nil
}

// rejectNonCanonicalPath enforces that every tar entry name is exactly its
// canonical form: no leading slash, no `.` / `..` segments, no double
// slashes. Combined with the strict-shape check in tarIDFromPath, this
// makes every accepted entry's id unambiguously derivable from its name.
func rejectNonCanonicalPath(name string) error {
	if name == "" {
		return errors.New("empty tar entry name")
	}
	if strings.HasPrefix(name, "/") {
		return fmt.Errorf("absolute tar entry name %q (malformed archive)", name)
	}
	if path.Clean(name) != name {
		return fmt.Errorf("non-canonical tar entry name %q (malformed archive)", name)
	}
	for _, seg := range strings.Split(name, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("malformed tar entry name %q", name)
		}
	}
	return nil
}

// getOrInitPending returns the pending entry for a table id, creating it on
// first reference. Returns nil if the id is not in the manifest header —
// that's a structural error and the caller should surface it.
func getOrInitPending(
	pending map[string]*pendingTable,
	tableID string,
	header chmanifest.Header,
) *pendingTable {
	if p, ok := pending[tableID]; ok {
		return p
	}
	for _, t := range header.Tables {
		if t.ID == tableID {
			p := &pendingTable{
				tableID:   tableID,
				tableName: t.Name,
			}
			pending[tableID] = p
			return p
		}
	}
	return nil
}

func setsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

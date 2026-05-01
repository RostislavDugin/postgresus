// Package chmanifest defines the on-disk manifest schema for ClickHouse backup
// archives. Both the create-backup and restore-backup use cases marshal/unmarshal
// these types; placing them here avoids an import cycle between the two.
//
// The archive is a single tar file with this layout (all paths use forward slashes):
//
//	manifest.header.json     — written FIRST, planning metadata only
//	db.sql                   — informational source CREATE DATABASE
//	tables/<id>/ddl.sql      — original SHOW CREATE TABLE per data table
//	tables/<id>/restore.sql  — DDL with ON CLUSTER + UUID stripped
//	tables/<id>/native.meta.json — written BEFORE native, holds size + sha256 + columns
//	tables/<id>/native       — raw `SELECT ... FORMAT Native` bytes
//	mvs/<id>/ddl.sql         — original DDL for TO-form materialized views
//	mvs/<id>/restore.sql     — cleaned DDL for the MV
//	manifest.footer.json     — written LAST with aggregate sizes and per-table cross-check
//
// IDs are opaque (e.g. "t-<sha8>" / "mv-<sha8>") to avoid path-traversal risk
// from user-controlled table names; the manifest carries the id↔name mapping.
package chmanifest

import "time"

// SchemaVersion is incremented on any incompatible change to the on-disk layout.
const SchemaVersion = 1

// Header is the first tar entry. It carries enough metadata for the restore use
// case to plan the restore (table list, ordering, MV target tables) without
// having to read the full archive first. Per-table sizes/hashes/columns live in
// NativeMeta sidecars and the Footer — they are unknown when the header is
// written.
type Header struct {
	SchemaVersion     int                `json:"schemaVersion"`
	CreatedAt         time.Time          `json:"createdAt"`
	ClickhouseVersion string             `json:"clickhouseVersion"`
	SourceDatabase    string             `json:"sourceDatabase"`
	SourceDBEngine    string             `json:"sourceDatabaseEngine"`
	Tables            []TableHeaderEntry `json:"tables"`
	MaterializedViews []MVHeaderEntry    `json:"materializedViews"`
	Flags             Flags              `json:"flags"`
	Notes             string             `json:"notes"`
}

// TableHeaderEntry describes a data table planned for backup. Columns is
// populated by enumeration but tagged json:"-" because it is authoritatively
// stored in the per-table NativeMeta sidecar; carrying it here would duplicate
// data and risk drift.
type TableHeaderEntry struct {
	ID        string   `json:"id"`
	Database  string   `json:"database"`
	Name      string   `json:"name"`
	Engine    string   `json:"engine"`
	DependsOn []string `json:"dependsOn,omitempty"`
	IsMV      bool     `json:"isMaterializedView"`

	Columns []string `json:"-"`
}

// MVHeaderEntry describes a TO-form materialized view. Implicit-storage MVs are
// rejected at pre-flight and never reach this struct.
type MVHeaderEntry struct {
	ID       string `json:"id"`
	Database string `json:"database"`
	Name     string `json:"name"`
	ToTable  string `json:"toTable"`
	DDLHash  string `json:"ddlHash"`
}

// Flags is reserved for future schema-level toggles (e.g. strict mode).
type Flags struct {
	StrictMode bool `json:"strictMode"`
}

// NativeMeta is the per-table sidecar written as `tables/<id>/native.meta.json`
// BEFORE the corresponding `tables/<id>/native` payload. The restore use case
// reads it, validates the subsequent native stream against NativeBytes + SHA256,
// and uses Columns to build the INSERT FORMAT Native column list.
type NativeMeta struct {
	Columns     []string `json:"columns"`
	NativeBytes int64    `json:"nativeBytes"`
	SHA256      string   `json:"sha256"`
	RowCount    int64    `json:"rowCount"`
	DDLHash     string   `json:"ddlHash"`
}

// Footer is the last tar entry. Status is "OK" only when every preceding entry
// was written successfully; mid-stream errors abort the pipeline before the
// footer is produced, leaving the storage object truncated.
type Footer struct {
	Tables      []TableFooterEntry `json:"tables"`
	TotalBytes  int64              `json:"totalBytes"`
	CompletedAt time.Time          `json:"completedAt"`
	Status      string             `json:"status"`
}

// TableFooterEntry is a redundant cross-check of NativeMeta. ID matches
// Header.Tables[].ID and the tar path prefix tables/<id>/.
type TableFooterEntry struct {
	ID          string `json:"id"`
	Database    string `json:"database"`
	Name        string `json:"name"`
	NativeBytes int64  `json:"nativeBytes"`
	SHA256      string `json:"sha256"`
	RowCount    int64  `json:"rowCount"`
}

package usecases_physical_postgresql

import physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"

// codecFallbackOrder is the server-side compression preference, highest ratio
// first. The real backup IS the probe: each attempt runs pg_basebackup, and a
// "compression is not supported by this build" rejection (raised pre-stream, so
// it costs only a sub-second round-trip) advances to the next codec. `none`
// needs no library and never raises that error, so the loop always terminates.
var codecFallbackOrder = []physical_enums.PhysicalBackupCompression{
	physical_enums.PhysicalBackupCompressionZstd,
	physical_enums.PhysicalBackupCompressionGzip,
	physical_enums.PhysicalBackupCompressionNone,
}

// compressFlag maps a codec to pg_basebackup's --compress value. Compression is
// server-side so only ~1/3 of the bytes cross the PG->Databasus link (ADR-0012);
// gzip:6 is the balanced analogue of zstd:5, and none is the no-library floor.
func compressFlag(codec physical_enums.PhysicalBackupCompression) string {
	switch codec {
	case physical_enums.PhysicalBackupCompressionGzip:
		return "server-gzip:6"

	case physical_enums.PhysicalBackupCompressionNone:
		return "none"

	default:
		return "server-zstd:5"
	}
}

package usecases_physical_postgresql

// Storage object-name suffixes for the sidecars written next to a backup
// artifact. The artifact key itself is extension-less (see buildObjectName);
// each sidecar appends one of these to it.
const (
	metadataSuffix = ".metadata"
	manifestSuffix = ".manifest"
)

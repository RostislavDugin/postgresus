package postgresql_physical

type replicationSettings struct {
	walLevel            string
	summarizeWal        string
	maxWalSenders       int
	maxReplicationSlots int
}

type ReplicationOnlyUser struct {
	Username                     string
	Password                     string
	IsForcedWalRotationAvailable bool
}

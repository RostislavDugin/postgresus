package postgresql_physical

type replicationSettings struct {
	walLevel            string
	summarizeWal        string
	maxWalSenders       int
	maxReplicationSlots int
}

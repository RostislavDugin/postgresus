package disk

type DiskUsage struct {
	TotalSpaceBytes int64 `json:"totalSpaceBytes"`
	UsedSpaceBytes  int64 `json:"usedSpaceBytes"`
	FreeSpaceBytes  int64 `json:"freeSpaceBytes"`
}

package disk

import (
	"databasus-backend/internal/config"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/shirou/gopsutil/v4/disk"
)

type DiskService struct{}

func (s *DiskService) GetDiskUsage() (*DiskUsage, error) {
	platform := s.detectPlatform()

	var path string

	if platform == PlatformWindows {
		path = "C:\\"
	} else {
		// Use databasus-data folder location for Linux (Docker)
		cfg := config.GetEnv()
		path = filepath.Dir(cfg.DataFolder) // Gets /databasus-data from /databasus-data/backups
	}

	diskUsage, err := disk.Usage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage for path %s: %w", path, err)
	}

	return &DiskUsage{
		Platform:        platform,
		TotalSpaceBytes: int64(diskUsage.Total),
		UsedSpaceBytes:  int64(diskUsage.Used),
		FreeSpaceBytes:  int64(diskUsage.Free),
	}, nil
}

func (s *DiskService) detectPlatform() Platform {
	switch runtime.GOOS {
	case "windows":
		return PlatformWindows
	default:
		return PlatformLinux
	}
}

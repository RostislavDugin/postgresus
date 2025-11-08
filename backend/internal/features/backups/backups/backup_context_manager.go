package backups

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

type BackupContextManager struct {
	mu          sync.RWMutex
	cancelFuncs map[uuid.UUID]context.CancelFunc
}

func NewBackupContextManager() *BackupContextManager {
	return &BackupContextManager{
		cancelFuncs: make(map[uuid.UUID]context.CancelFunc),
	}
}

func (m *BackupContextManager) RegisterBackup(backupID uuid.UUID, cancelFunc context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelFuncs[backupID] = cancelFunc
}

func (m *BackupContextManager) CancelBackup(backupID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cancelFunc, exists := m.cancelFuncs[backupID]
	if !exists {
		return errors.New("backup is not in progress or already completed")
	}

	cancelFunc()
	delete(m.cancelFuncs, backupID)

	return nil
}

func (m *BackupContextManager) UnregisterBackup(backupID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancelFuncs, backupID)
}

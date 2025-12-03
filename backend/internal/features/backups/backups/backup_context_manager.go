package backups

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type BackupContextManager struct {
	mu               sync.RWMutex
	cancelFuncs      map[uuid.UUID]context.CancelFunc
	cancelledBackups map[uuid.UUID]bool
}

func NewBackupContextManager() *BackupContextManager {
	return &BackupContextManager{
		cancelFuncs:      make(map[uuid.UUID]context.CancelFunc),
		cancelledBackups: make(map[uuid.UUID]bool),
	}
}

func (m *BackupContextManager) RegisterBackup(backupID uuid.UUID, cancelFunc context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelFuncs[backupID] = cancelFunc
	delete(m.cancelledBackups, backupID)
}

func (m *BackupContextManager) CancelBackup(backupID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelledBackups[backupID] {
		return nil
	}

	cancelFunc, exists := m.cancelFuncs[backupID]
	if exists {
		cancelFunc()
		delete(m.cancelFuncs, backupID)
	}

	m.cancelledBackups[backupID] = true

	return nil
}

func (m *BackupContextManager) IsCancelled(backupID uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cancelledBackups[backupID]
}

func (m *BackupContextManager) UnregisterBackup(backupID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancelFuncs, backupID)
	delete(m.cancelledBackups, backupID)
}

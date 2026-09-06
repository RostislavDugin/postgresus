package cache

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	payload   []byte
	expiresAt time.Time
	sizeBytes int64
}

type MemoryStore struct {
	mutex              sync.Mutex
	entries            map[string]memoryEntry
	payloadBytes       int64
	payloadBudgetBytes int64
	getCurrentTime     func() time.Time
}

func NewMemoryStore(payloadBudgetBytes int64) *MemoryStore {
	return newMemoryStore(payloadBudgetBytes, func() time.Time {
		return time.Now().UTC()
	})
}

func newMemoryStore(payloadBudgetBytes int64, getCurrentTime func() time.Time) *MemoryStore {
	if payloadBudgetBytes <= 0 {
		panic("cache payload budget must be positive")
	}

	return &MemoryStore{
		entries:            make(map[string]memoryEntry),
		payloadBudgetBytes: payloadBudgetBytes,
		getCurrentTime:     getCurrentTime,
	}
}

func (s *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry, isFound := s.getLiveEntry(key, s.getCurrentTime())
	if !isFound {
		return nil, false, nil
	}

	return append([]byte(nil), entry.payload...), true, nil
}

func (s *MemoryStore) Set(ctx context.Context, entry Entry) error {
	if err := validateWrite(ctx, entry); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	currentTime := s.getCurrentTime()

	return s.store(entry, currentTime)
}

func (s *MemoryStore) CreateIfAbsent(ctx context.Context, entry Entry) (bool, error) {
	if err := validateWrite(ctx, entry); err != nil {
		return false, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	currentTime := s.getCurrentTime()
	if _, isFound := s.getLiveEntry(entry.Key, currentTime); isFound {
		return false, nil
	}

	if err := s.store(entry, currentTime); err != nil {
		return false, err
	}

	return true, nil
}

func (s *MemoryStore) ReadAndDelete(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry, isFound := s.getLiveEntry(key, s.getCurrentTime())
	if !isFound {
		return nil, false, nil
	}

	s.remove(key, entry)

	return append([]byte(nil), entry.payload...), true, nil
}

func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if entry, isFound := s.entries[key]; isFound {
		s.remove(key, entry)
	}

	return nil
}

func (s *MemoryStore) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.entries = make(map[string]memoryEntry)
	s.payloadBytes = 0

	return nil
}

func validateWrite(ctx context.Context, entry Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if entry.Lifetime <= 0 {
		return ErrInvalidLifetime
	}

	return nil
}

func (s *MemoryStore) getLiveEntry(key string, currentTime time.Time) (memoryEntry, bool) {
	entry, isFound := s.entries[key]
	if !isFound {
		return memoryEntry{}, false
	}

	if !currentTime.Before(entry.expiresAt) {
		s.remove(key, entry)

		return memoryEntry{}, false
	}

	return entry, true
}

func (s *MemoryStore) reclaimExpired(currentTime time.Time) {
	for key, entry := range s.entries {
		if !currentTime.Before(entry.expiresAt) {
			s.remove(key, entry)
		}
	}
}

func (s *MemoryStore) store(entry Entry, currentTime time.Time) error {
	entrySizeBytes := int64(len(entry.Key) + len(entry.Payload))
	previousSizeBytes := s.getStoredEntrySizeBytes(entry.Key)

	if s.payloadBytes-previousSizeBytes+entrySizeBytes > s.payloadBudgetBytes {
		s.reclaimExpired(currentTime)
		previousSizeBytes = s.getStoredEntrySizeBytes(entry.Key)

		if s.payloadBytes-previousSizeBytes+entrySizeBytes > s.payloadBudgetBytes {
			return ErrPayloadBudgetReached
		}
	}

	payload := append([]byte(nil), entry.Payload...)
	s.entries[entry.Key] = memoryEntry{
		payload:   payload,
		expiresAt: currentTime.Add(entry.Lifetime),
		sizeBytes: entrySizeBytes,
	}
	s.payloadBytes = s.payloadBytes - previousSizeBytes + entrySizeBytes

	return nil
}

func (s *MemoryStore) getStoredEntrySizeBytes(key string) int64 {
	entry, isFound := s.entries[key]
	if !isFound {
		return 0
	}

	return entry.sizeBytes
}

func (s *MemoryStore) remove(key string, entry memoryEntry) {
	delete(s.entries, key)
	s.payloadBytes -= entry.sizeBytes
}

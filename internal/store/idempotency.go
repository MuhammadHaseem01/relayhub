package store

import (
	"sync"
	"time"
)

// IdempotencyRecord stores the result of an idempotent operation
type IdempotencyRecord struct {
	StatusCode int
	Body       []byte
	CreatedAt  time.Time
	InProgress bool
}

// IdempotencyStore defines the contract for checking and storing idempotency keys
type IdempotencyStore interface {
	// GetOrCreate checks if a key exists. If not, it creates an InProgress record.
	// Returns the record and a boolean indicating if the key ALREADY existed.
	GetOrCreate(key string) (IdempotencyRecord, bool)
	
	// Save finalizes an InProgress record with the actual response.
	Save(key string, statusCode int, body []byte)
}

// InMemoryIdempotencyStore is a simple thread-safe map implementation
type InMemoryIdempotencyStore struct {
	mu    sync.Mutex
	store map[string]IdempotencyRecord
	ttl   time.Duration
}

// NewInMemoryIdempotencyStore creates a new store with a 5-minute TTL
func NewInMemoryIdempotencyStore() *InMemoryIdempotencyStore {
	return &InMemoryIdempotencyStore{
		store: make(map[string]IdempotencyRecord),
		ttl:   5 * time.Minute,
	}
}

func (s *InMemoryIdempotencyStore) GetOrCreate(key string) (IdempotencyRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.store[key]
	
	// Check expiry
	if exists && time.Since(record.CreatedAt) > s.ttl {
		// Key expired, delete it and treat as if it didn't exist
		delete(s.store, key)
		exists = false
	}

	if exists {
		return record, true
	}

	// Create new in-progress record
	newRecord := IdempotencyRecord{
		CreatedAt:  time.Now(),
		InProgress: true,
	}
	s.store[key] = newRecord
	return newRecord, false
}

func (s *InMemoryIdempotencyStore) Save(key string, statusCode int, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record, exists := s.store[key]; exists {
		record.StatusCode = statusCode
		record.Body = body
		record.InProgress = false
		s.store[key] = record
	}
}

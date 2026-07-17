package store

import (
	"testing"
	"time"
)

func TestInMemoryIdempotencyStore(t *testing.T) {
	s := NewInMemoryIdempotencyStore()
	key := "test-key-1"

	// 1. Initial request
	record, exists := s.GetOrCreate(key)
	if exists {
		t.Fatalf("expected key to not exist on first call")
	}
	if !record.InProgress {
		t.Fatalf("expected new record to be marked as InProgress")
	}

	// 2. Concurrent request (race condition)
	record2, exists2 := s.GetOrCreate(key)
	if !exists2 {
		t.Fatalf("expected key to exist on second call")
	}
	if !record2.InProgress {
		t.Fatalf("expected concurrent request to see InProgress=true")
	}

	// 3. Complete the request
	s.Save(key, 200, []byte(`{"status":"ok"}`))

	// 4. Subsequent request after completion
	record3, exists3 := s.GetOrCreate(key)
	if !exists3 {
		t.Fatalf("expected key to exist after save")
	}
	if record3.InProgress {
		t.Fatalf("expected record to no longer be InProgress")
	}
	if record3.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", record3.StatusCode)
	}
	if string(record3.Body) != `{"status":"ok"}` {
		t.Fatalf("expected body to match saved body, got %s", string(record3.Body))
	}
}

func TestInMemoryIdempotencyStore_Expiry(t *testing.T) {
	s := NewInMemoryIdempotencyStore()
	// artificially short TTL for test
	s.ttl = 10 * time.Millisecond
	key := "test-key-2"

	s.GetOrCreate(key)
	s.Save(key, 200, []byte("ok"))

	// Wait for expiry
	time.Sleep(15 * time.Millisecond)

	// Should be treated as new request
	record, exists := s.GetOrCreate(key)
	if exists {
		t.Fatalf("expected key to be expired and not exist")
	}
	if !record.InProgress {
		t.Fatalf("expected new record to be InProgress")
	}
}

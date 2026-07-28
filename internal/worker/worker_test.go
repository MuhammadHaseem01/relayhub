package worker_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"relayhub/internal/dispatch"
	"relayhub/internal/providers"
	"relayhub/internal/queue"
	"relayhub/internal/worker"
)

type mockSender struct {
	name    string
	calls   atomic.Int32
	failErr error
}

func (m *mockSender) Name() string { return m.name }
func (m *mockSender) Send(_, _ string) error {
	m.calls.Add(1)
	return m.failErr
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func noopRetry(fn func() error, _ int, _ *slog.Logger) (int, error) {
	return 1, fn()
}

func setupQueue(t *testing.T) (*queue.Queue, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := queue.NewFromClient(rdb)
	if err := q.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	return q, rdb, mr
}

func makeExecutor(sender *mockSender) *dispatch.Executor {
	return &dispatch.Executor{
		Providers:   map[string]providers.Sender{sender.name: sender},
		Retry:       noopRetry,
		MaxAttempts: 1,
		Store:       nil,
		Dispatcher:  nil,
		Logger:      noopLogger(),
	}
}

func TestWorker_ProcessesJob(t *testing.T) {
	q, rdb, _ := setupQueue(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sender := &mockSender{name: "email"}
	exec := makeExecutor(sender)

	pool := worker.New(worker.Params{
		WorkerCount: 1,
		Queue:       q,
		Executor:    exec,
		Logger:      noopLogger(),
	})
	go pool.Run(ctx)

	job := queue.NotificationJob{
		RequestID: "worker-test-001",
		TenantID:  "tenant-abc",
		Channel:   "email",
		Recipient: "bob@example.com",
		Message:   "hello from worker test",
	}
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if sender.calls.Load() >= 1 {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if sender.calls.Load() == 0 {
		t.Fatal("expected provider.Send to be called, but it wasn't within 3s")
	}

	time.Sleep(100 * time.Millisecond)
	cancel() // stop pool

	pending, err := rdb.XPending(context.Background(), queue.StreamName, queue.ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending: %v", err)
	}
	if pending.Count != 0 {
		t.Errorf("expected 0 pending after ACK, got %d", pending.Count)
	}
}

func TestWorker_FailedSend_StillAcks(t *testing.T) {
	q, rdb, _ := setupQueue(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sender := &mockSender{name: "email", failErr: errors.New("smtp timeout")}
	exec := makeExecutor(sender)

	pool := worker.New(worker.Params{
		WorkerCount: 1,
		Queue:       q,
		Executor:    exec,
		Logger:      noopLogger(),
	})
	go pool.Run(ctx)

	if err := q.Enqueue(ctx, queue.NotificationJob{
		RequestID: "fail-test-001",
		TenantID:  "t1",
		Channel:   "email",
		Recipient: "x@example.com",
		Message:   "will fail",
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if sender.calls.Load() >= 1 {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	cancel()

	pending, _ := rdb.XPending(context.Background(), queue.StreamName, queue.ConsumerGroup).Result()
	if pending.Count != 0 {
		t.Errorf("expected 0 pending after failed send, got %d", pending.Count)
	}
}

func TestWorker_FastEnqueue_NoProviderBlock(t *testing.T) {
	q, _, _ := setupQueue(t)
	ctx := context.Background()

	start := time.Now()
	for i := range 20 {
		job := queue.NotificationJob{
			RequestID: "fast-" + string(rune('a'+i)),
			TenantID:  "tenant",
			Channel:   "email",
			Recipient: "fast@example.com",
			Message:   "fire and forget",
		}
		if err := q.Enqueue(ctx, job); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("20 enqueues took %v; expected <200ms (should be non-blocking)", elapsed)
	}
}

func TestReclaimer_ClaimsOrphanedMessages(t *testing.T) {
	q, _, mr := setupQueue(t)
	ctx := context.Background()

	if err := q.Enqueue(ctx, queue.NotificationJob{
		RequestID: "orphan-req-001",
		TenantID:  "t1",
		Channel:   "email",
		Recipient: "carol@example.com",
		Message:   "orphaned",
	}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if msgs, err := q.Dequeue(ctx, "crashed-worker", 1, 0); err != nil || len(msgs) != 1 {
		t.Fatalf("initial Dequeue: err=%v len=%d", err, len(msgs))
	}

	mr.SetTime(time.Now().Add(10 * time.Minute))

	reclaimed, err := q.Reclaim(ctx, "reclaimer", time.Second, 10)
	if err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	if len(reclaimed) != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", len(reclaimed))
	}
	if reclaimed[0].Job.RequestID != "orphan-req-001" {
		t.Errorf("unexpected RequestID: %q", reclaimed[0].Job.RequestID)
	}
}

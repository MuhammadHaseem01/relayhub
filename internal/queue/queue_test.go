package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"relayhub/internal/queue"
)

func newTestQueue(t *testing.T) (*queue.Queue, *miniredis.Miniredis) {
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
	return q, mr
}

func sampleJob(id string) queue.NotificationJob {
	return queue.NotificationJob{
		RequestID:      id,
		TenantID:       "tenant-123",
		Channel:        "email",
		Recipient:      "alice@example.com",
		Message:        "hello from test",
		IdempotencyKey: "idem-" + id,
	}
}

func TestEnsureGroup_Idempotent(t *testing.T) {
	q, _ := newTestQueue(t)
	if err := q.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("second EnsureGroup: %v", err)
	}
}

func TestEnqueue_Dequeue_RoundTrip(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()
	job := sampleJob("req-001")

	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	msgs, err := q.Dequeue(ctx, "worker-0", 1, 0)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	got := msgs[0].Job
	if got.RequestID != job.RequestID {
		t.Errorf("RequestID: want %q, got %q", job.RequestID, got.RequestID)
	}
	if got.TenantID != job.TenantID {
		t.Errorf("TenantID: want %q, got %q", job.TenantID, got.TenantID)
	}
	if got.Channel != job.Channel {
		t.Errorf("Channel: want %q, got %q", job.Channel, got.Channel)
	}
	if got.Recipient != job.Recipient {
		t.Errorf("Recipient: want %q, got %q", job.Recipient, got.Recipient)
	}
	if got.Message != job.Message {
		t.Errorf("Message: want %q, got %q", job.Message, got.Message)
	}
}

func TestAck_RemovesFromPending(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := queue.NewFromClient(rdb)
	ctx := context.Background()

	if err := q.EnsureGroup(ctx); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	if err := q.Enqueue(ctx, sampleJob("req-002")); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	msgs, err := q.Dequeue(ctx, "worker-0", 1, 0)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("Dequeue: err=%v len=%d", err, len(msgs))
	}
	streamID := msgs[0].ID

	// Before ACK: XPENDING should report 1 pending message
	pendingBefore, err := rdb.XPending(ctx, queue.StreamName, queue.ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending before ACK: %v", err)
	}
	if pendingBefore.Count != 1 {
		t.Errorf("expected 1 pending before ACK, got %d", pendingBefore.Count)
	}

	if err := q.Ack(ctx, streamID); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	// After ACK: pending count should drop to 0
	pendingAfter, err := rdb.XPending(ctx, queue.StreamName, queue.ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("XPending after ACK: %v", err)
	}
	if pendingAfter.Count != 0 {
		t.Errorf("expected 0 pending after ACK, got %d", pendingAfter.Count)
	}
}

func TestDequeue_EmptyStream_NoBlock(t *testing.T) {
	q, _ := newTestQueue(t)

	// Use a short-deadline context so XREADGROUP returns quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Stream is empty; Dequeue should return nil, nil (context cancellation
	// is treated as "no messages" when the stream is genuinely empty).
	msgs, err := q.Dequeue(ctx, "worker-0", 1, 100*time.Millisecond)
	// Context deadline errors are expected when there are no messages.
	if err != nil && ctx.Err() == nil {
		t.Fatalf("unexpected error (not context-related): %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestReclaim_RecoversIdleMessages(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := queue.NewFromClient(rdb)
	ctx := context.Background()

	if err := q.EnsureGroup(ctx); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}
	if err := q.Enqueue(ctx, sampleJob("req-003")); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Worker-0 claims the message but never ACKs it (simulates crash).
	msgs, err := q.Dequeue(ctx, "worker-0", 1, 0)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("Dequeue: err=%v len=%d", err, len(msgs))
	}

	// Advance miniredis internal clock by 10 minutes so the message appears idle.
	mr.SetTime(time.Now().Add(10 * time.Minute))

	// Reclaimer should find and claim it (minIdleTime=1s so our 10min easily qualifies).
	reclaimed, err := q.Reclaim(ctx, "reclaimer", time.Second, 10)
	if err != nil {
		t.Fatalf("Reclaim: %v", err)
	}
	if len(reclaimed) != 1 {
		t.Errorf("expected 1 reclaimed message, got %d", len(reclaimed))
		return
	}
	if reclaimed[0].Job.RequestID != "req-003" {
		t.Errorf("reclaimed wrong job: %q", reclaimed[0].Job.RequestID)
	}
}

func TestMultipleJobs_OrderPreserved(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	ids := []string{"req-a", "req-b", "req-c"}
	for _, id := range ids {
		if err := q.Enqueue(ctx, sampleJob(id)); err != nil {
			t.Fatalf("Enqueue %s: %v", id, err)
		}
	}

	msgs, err := q.Dequeue(ctx, "worker-0", 3, 0)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	for i, id := range ids {
		if msgs[i].Job.RequestID != id {
			t.Errorf("msg[%d]: want RequestID=%q, got %q", i, id, msgs[i].Job.RequestID)
		}
	}
}

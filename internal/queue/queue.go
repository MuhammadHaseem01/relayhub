package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StreamName = "relayhub:notifications"

	ConsumerGroup = "relayhub-workers"

	maxStreamLen = 10_000
)

type NotificationJob struct {
	RequestID        string
	TenantID         string
	Channel          string
	Recipient        string
	Message          string
	IdempotencyKey   string
	DiscordRecipient string
	EmailRecipient   string
}

type Message struct {
	ID  string
	Job NotificationJob
}

type Queue struct {
	rdb *redis.Client
}

func New(redisURL string) (*Queue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("queue: invalid REDIS_URL %q: %w", redisURL, err)
	}
	return &Queue{rdb: redis.NewClient(opts)}, nil
}

func NewFromClient(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

func (q *Queue) Ping(ctx context.Context) error {
	if err := q.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("queue: Redis unreachable: %w", err)
	}
	return nil
}

func (q *Queue) Close() error { return q.rdb.Close() }

func (q *Queue) EnsureGroup(ctx context.Context) error {
	err := q.rdb.XGroupCreateMkStream(ctx, StreamName, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("queue: failed to create consumer group: %w", err)
	}
	return nil
}

func (q *Queue) Enqueue(ctx context.Context, job NotificationJob) error {
	err := q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		MaxLen: maxStreamLen,
		Approx: true,
		Values: jobToMap(job),
	}).Err()
	if err != nil {
		return fmt.Errorf("queue: failed to enqueue job %q: %w", job.RequestID, err)
	}
	return nil
}

func (q *Queue) Dequeue(
	ctx context.Context,
	consumerName string,
	count int64,
	blockDuration time.Duration,
) ([]Message, error) {
	entries, err := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{StreamName, ">"},
		Count:    count,
		Block:    blockDuration,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("queue: dequeue error: %w", err)
	}
	if len(entries) == 0 {
		return nil, nil
	}
	msgs := make([]Message, 0, len(entries[0].Messages))
	for _, xm := range entries[0].Messages {
		msgs = append(msgs, Message{ID: xm.ID, Job: jobFromMap(xm.Values)})
	}
	return msgs, nil
}

func (q *Queue) Ack(ctx context.Context, streamID string) error {
	if err := q.rdb.XAck(ctx, StreamName, ConsumerGroup, streamID).Err(); err != nil {
		return fmt.Errorf("queue: failed to ack %q: %w", streamID, err)
	}
	return nil
}

// Reclaim re-assigns up to count messages that have been pending (unacknowledged)
// for longer than minIdleTime to consumerName.  This recovers jobs that were
// claimed by a worker that crashed before ACKing.
func (q *Queue) Reclaim(
	ctx context.Context,
	consumerName string,
	minIdleTime time.Duration,
	count int64,
) ([]Message, error) {
	// XAutoClaim returns (messages, nextStartID, error)
	msgsRaw, _, err := q.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   StreamName,
		Group:    ConsumerGroup,
		Consumer: consumerName,
		MinIdle:  minIdleTime,
		Start:    "0-0",
		Count:    count,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("queue: XAUTOCLAIM error: %w", err)
	}
	msgs := make([]Message, 0, len(msgsRaw))
	for _, xm := range msgsRaw {
		msgs = append(msgs, Message{ID: xm.ID, Job: jobFromMap(xm.Values)})
	}
	return msgs, nil
}

func jobToMap(job NotificationJob) map[string]any {
	return map[string]any{
		"request_id":        job.RequestID,
		"tenant_id":         job.TenantID,
		"channel":           job.Channel,
		"recipient":         job.Recipient,
		"message":           job.Message,
		"idempotency_key":   job.IdempotencyKey,
		"discord_recipient": job.DiscordRecipient,
		"email_recipient":   job.EmailRecipient,
	}
}

func jobFromMap(vals map[string]any) NotificationJob {
	str := func(key string) string {
		if v, ok := vals[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	return NotificationJob{
		RequestID:        str("request_id"),
		TenantID:         str("tenant_id"),
		Channel:          str("channel"),
		Recipient:        str("recipient"),
		Message:          str("message"),
		IdempotencyKey:   str("idempotency_key"),
		DiscordRecipient: str("discord_recipient"),
		EmailRecipient:   str("email_recipient"),
	}
}

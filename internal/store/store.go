package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRecord struct {
	ID           int64     `json:"id"`
	RequestID    string    `json:"request_id"`
	Recipient    string    `json:"recipient"`
	Channel      string    `json:"channel"`
	Message      string    `json:"message"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	Attempts          int       `json:"attempts"`
	FallbackUsed      bool      `json:"fallback_used"`
	IdempotencyKey    string    `json:"idempotency_key"`
	WasCachedResponse bool      `json:"was_cached_response"`
	CreatedAt         time.Time `json:"created_at"`
}
type Store struct {
	pool *pgxpool.Pool
}

func New(databaseURL string) (*Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: failed to parse DATABASE_URL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("store: postgres unreachable — is it running? %w", err)
	}

	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("store: schema migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notifications (
			id            BIGSERIAL    PRIMARY KEY,
			request_id    TEXT         NOT NULL,
			recipient     TEXT         NOT NULL,
			channel       TEXT         NOT NULL,
			message       TEXT         NOT NULL,
			status        TEXT         NOT NULL,
			error_message       TEXT         NOT NULL DEFAULT '',
			attempts            INT          NOT NULL DEFAULT 1,
			fallback_used       BOOLEAN      NOT NULL DEFAULT false,
			idempotency_key     TEXT         DEFAULT '',
			was_cached_response BOOLEAN      NOT NULL DEFAULT false,
			created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_notifications_request_id
			ON notifications (request_id);

		CREATE INDEX IF NOT EXISTS idx_notifications_created_at
			ON notifications (created_at DESC);
	`)
	return err
}

func (s *Store) LogNotification(ctx context.Context, rec NotificationRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications
			(request_id, recipient, channel, message, status, error_message, attempts, fallback_used, idempotency_key, was_cached_response)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, rec.RequestID, rec.Recipient, rec.Channel, rec.Message, rec.Status, rec.ErrorMessage, rec.Attempts, rec.FallbackUsed, rec.IdempotencyKey, rec.WasCachedResponse)
	if err != nil {
		return fmt.Errorf("store: failed to log notification: %w", err)
	}
	return nil
}

func (s *Store) GetLogs(ctx context.Context, limit int) ([]NotificationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, request_id, recipient, channel, message, status, error_message, attempts, fallback_used, idempotency_key, was_cached_response, created_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: failed to query logs: %w", err)
	}
	defer rows.Close()

	var records []NotificationRecord
	for rows.Next() {
		var r NotificationRecord
		if err := rows.Scan(
			&r.ID, &r.RequestID, &r.Recipient, &r.Channel,
			&r.Message, &r.Status, &r.ErrorMessage, &r.Attempts, &r.FallbackUsed, &r.IdempotencyKey, &r.WasCachedResponse, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: row scan error: %w", err)
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

func (s *Store) Close() {
	s.pool.Close()
}

package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationRecord represents a single delivery attempt persisted in Postgres.
type NotificationRecord struct {
	ID           int64     `json:"id"`
	RequestID    string    `json:"request_id"`
	Recipient    string    `json:"recipient"`
	Channel      string    `json:"channel"`
	Message      string    `json:"message"`
	Status       string    `json:"status"`        // "delivered" | "failed"
	ErrorMessage string    `json:"error_message"` // empty string on success
	CreatedAt    time.Time `json:"created_at"`
}

// Store wraps the PostgreSQL connection pool and exposes typed operations.
type Store struct {
	pool *pgxpool.Pool
}

// New opens a connection pool to the given PostgreSQL DSN and runs the
// embedded schema migration (idempotent — safe to call on every startup).
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

// migrate runs the embedded DDL. All statements use IF NOT EXISTS so this is
// fully idempotent and safe to execute on every application startup.
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notifications (
			id            BIGSERIAL    PRIMARY KEY,
			request_id    TEXT         NOT NULL,
			recipient     TEXT         NOT NULL,
			channel       TEXT         NOT NULL,
			message       TEXT         NOT NULL,
			status        TEXT         NOT NULL,
			error_message TEXT         NOT NULL DEFAULT '',
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_notifications_request_id
			ON notifications (request_id);

		CREATE INDEX IF NOT EXISTS idx_notifications_created_at
			ON notifications (created_at DESC);
	`)
	return err
}

// LogNotification inserts a delivery record. Called after every /notify attempt.
func (s *Store) LogNotification(ctx context.Context, rec NotificationRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications
			(request_id, recipient, channel, message, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, rec.RequestID, rec.Recipient, rec.Channel, rec.Message, rec.Status, rec.ErrorMessage)
	if err != nil {
		return fmt.Errorf("store: failed to log notification: %w", err)
	}
	return nil
}

// GetLogs returns the most recent notifications, newest first.
func (s *Store) GetLogs(ctx context.Context, limit int) ([]NotificationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, request_id, recipient, channel, message, status, error_message, created_at
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
			&r.Message, &r.Status, &r.ErrorMessage, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: row scan error: %w", err)
		}
		records = append(records, r)
	}

	return records, rows.Err()
}

// Close releases the connection pool. Call via defer in main.
func (s *Store) Close() {
	s.pool.Close()
}

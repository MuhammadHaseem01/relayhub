package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationRecord struct {
	ID                int64     `json:"id"`
	TenantID          string    `json:"tenant_id"`
	RequestID         string    `json:"request_id"`
	Recipient         string    `json:"recipient"`
	Channel           string    `json:"channel"`
	Message           string    `json:"message"`
	Status            string    `json:"status"`
	ErrorMessage      string    `json:"error_message"`
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
		-- ── Tenants ──────────────────────────────────────────────────────────
		CREATE TABLE IF NOT EXISTS tenants (
			id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			name       TEXT        NOT NULL,
			api_key    TEXT        NOT NULL,
			plan       TEXT        NOT NULL DEFAULT 'free',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_api_key
			ON tenants (api_key);

		-- ── Notifications ─────────────────────────────────────────────────────
		CREATE TABLE IF NOT EXISTS notifications (
			id                  BIGSERIAL    PRIMARY KEY,
			tenant_id           UUID         REFERENCES tenants(id),
			request_id          TEXT         NOT NULL,
			recipient           TEXT         NOT NULL,
			channel             TEXT         NOT NULL,
			message             TEXT         NOT NULL,
			status              TEXT         NOT NULL,
			error_message       TEXT         NOT NULL DEFAULT '',
			attempts            INT          NOT NULL DEFAULT 1,
			fallback_used       BOOLEAN      NOT NULL DEFAULT false,
			idempotency_key     TEXT         DEFAULT '',
			was_cached_response BOOLEAN      NOT NULL DEFAULT false,
			created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		);

		-- Safe backfill: add tenant_id to the table if it was created before
		-- Phase 2. Existing rows will have NULL tenant_id (acceptable).
		ALTER TABLE notifications
			ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id);

		CREATE INDEX IF NOT EXISTS idx_notifications_request_id
			ON notifications (request_id);

		CREATE INDEX IF NOT EXISTS idx_notifications_created_at
			ON notifications (created_at DESC);

		CREATE INDEX IF NOT EXISTS idx_notifications_tenant_id
			ON notifications (tenant_id);
	`)
	return err
}

func (s *Store) CreateTenant(ctx context.Context, name, apiKey string) (TenantRecord, error) {
	var rec TenantRecord
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, api_key)
		VALUES ($1, $2)
		RETURNING id, name, api_key, plan, created_at
	`, name, apiKey).Scan(&rec.ID, &rec.Name, &rec.APIKey, &rec.Plan, &rec.CreatedAt)
	if err != nil {
		return TenantRecord{}, fmt.Errorf("store: failed to create tenant: %w", err)
	}
	return rec, nil
}

func (s *Store) GetTenantByAPIKey(ctx context.Context, apiKey string) (TenantRecord, error) {
	var rec TenantRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, plan, created_at
		FROM tenants
		WHERE api_key = $1
	`, apiKey).Scan(&rec.ID, &rec.Name, &rec.APIKey, &rec.Plan, &rec.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TenantRecord{}, fmt.Errorf("store: tenant not found: %w", err)
		}
		return TenantRecord{}, fmt.Errorf("store: failed to lookup tenant: %w", err)
	}
	return rec, nil
}

func (s *Store) LogNotification(ctx context.Context, rec NotificationRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications
			(tenant_id, request_id, recipient, channel, message, status,
			 error_message, attempts, fallback_used, idempotency_key, was_cached_response)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, rec.TenantID, rec.RequestID, rec.Recipient, rec.Channel, rec.Message, rec.Status,
		rec.ErrorMessage, rec.Attempts, rec.FallbackUsed, rec.IdempotencyKey, rec.WasCachedResponse)
	if err != nil {
		return fmt.Errorf("store: failed to log notification: %w", err)
	}
	return nil
}

func (s *Store) GetLogs(ctx context.Context, tenantID string, limit int) ([]NotificationRecord, error) {
	if tenantID == "" {
		return []NotificationRecord{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, request_id, recipient, channel, message, status,
		       error_message, attempts, fallback_used, idempotency_key,
		       was_cached_response, created_at
		FROM notifications
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: failed to query logs: %w", err)
	}
	defer rows.Close()

	var records []NotificationRecord
	for rows.Next() {
		var r NotificationRecord
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.RequestID, &r.Recipient, &r.Channel,
			&r.Message, &r.Status, &r.ErrorMessage, &r.Attempts,
			&r.FallbackUsed, &r.IdempotencyKey, &r.WasCachedResponse, &r.CreatedAt,
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

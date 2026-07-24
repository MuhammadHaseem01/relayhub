package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDuplicateTemplate = errors.New("store: template name already exists for this tenant")
var ErrTemplateNotFound = errors.New("store: template not found")
var ErrNotificationNotFound = errors.New("store: notification not found")
var ErrNotificationAlreadySent = errors.New("store: notification already sent or cancelled")
var ErrTenantNotFound = errors.New("store: tenant not found")

type TenantRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	APIKey        string    `json:"api_key"`
	Plan          string    `json:"plan"`
	WebhookURL    string    `json:"webhook_url,omitempty"`
	WebhookSecret string    `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}

type WebhookDeliveryRecord struct {
	ID                    int64     `json:"id"`
	TenantID              string    `json:"tenant_id"`
	NotificationRequestID string    `json:"notification_request_id"`
	StatusCode            int       `json:"status_code"`
	Attempt               int       `json:"attempt"`
	Success               bool      `json:"success"`
	CreatedAt             time.Time `json:"created_at"`
}

type TemplateRecord struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NotificationRecord struct {
	ID                int64      `json:"id"`
	TenantID          string     `json:"tenant_id"`
	RequestID         string     `json:"request_id"`
	Recipient         string     `json:"recipient"`
	Channel           string     `json:"channel"`
	Message           string     `json:"message"`
	Status            string     `json:"status"`
	ErrorMessage      string     `json:"error_message"`
	Attempts          int        `json:"attempts"`
	FallbackUsed      bool       `json:"fallback_used"`
	IdempotencyKey    string     `json:"idempotency_key"`
	WasCachedResponse bool       `json:"was_cached_response"`
	CreatedAt         time.Time  `json:"created_at"`
	ScheduledFor      *time.Time `json:"scheduled_for,omitempty"`
	DiscordRecipient  string     `json:"discord_recipient,omitempty"`
	EmailRecipient    string     `json:"email_recipient,omitempty"`
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

		-- ── Scheduled-send columns (Phase 3 Step 2) ───────────────────────────
		ALTER TABLE notifications
			ADD COLUMN IF NOT EXISTS scheduled_for     TIMESTAMPTZ NULL,
			ADD COLUMN IF NOT EXISTS discord_recipient TEXT        NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS email_recipient   TEXT        NOT NULL DEFAULT '';

		-- Partial index: only index rows that still need to fire.
		CREATE INDEX IF NOT EXISTS idx_notifications_scheduled
			ON notifications (scheduled_for ASC)
			WHERE status = 'scheduled';

		-- ── Templates ──────────────────────────────────────────────────────────
		CREATE TABLE IF NOT EXISTS templates (
			id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id  UUID        NOT NULL REFERENCES tenants(id),
			name       TEXT        NOT NULL,
			body       TEXT        NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_templates_tenant_name
			ON templates (tenant_id, name);

		-- ── Webhook columns (Phase 3 Step 3) ────────────────────────────────────
		ALTER TABLE tenants
			ADD COLUMN IF NOT EXISTS webhook_url    TEXT NULL,
			ADD COLUMN IF NOT EXISTS webhook_secret TEXT NULL;

		-- ── Webhook delivery audit log ───────────────────────────────────────────
		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id                      BIGSERIAL    PRIMARY KEY,
			tenant_id               UUID         NOT NULL REFERENCES tenants(id),
			notification_request_id TEXT         NOT NULL,
			status_code             INT          NOT NULL DEFAULT 0,
			attempt                 INT          NOT NULL DEFAULT 1,
			success                 BOOLEAN      NOT NULL DEFAULT false,
			created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant_id
			ON webhook_deliveries (tenant_id, created_at DESC);
	`)
	return err
}

func (s *Store) CreateTemplate(ctx context.Context, tenantID, name, body string) (TemplateRecord, error) {
	var rec TemplateRecord
	err := s.pool.QueryRow(ctx, `
		INSERT INTO templates (tenant_id, name, body)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, name, body, created_at, updated_at
	`, tenantID, name, body).Scan(
		&rec.ID, &rec.TenantID, &rec.Name, &rec.Body, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return TemplateRecord{}, ErrDuplicateTemplate
		}
		return TemplateRecord{}, fmt.Errorf("store: failed to create template: %w", err)
	}
	return rec, nil
}

func (s *Store) GetTemplate(ctx context.Context, tenantID, name string) (TemplateRecord, error) {
	var rec TemplateRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, body, created_at, updated_at
		FROM templates
		WHERE tenant_id = $1 AND name = $2
	`, tenantID, name).Scan(
		&rec.ID, &rec.TenantID, &rec.Name, &rec.Body, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRecord{}, ErrTemplateNotFound
		}
		return TemplateRecord{}, fmt.Errorf("store: failed to get template: %w", err)
	}
	return rec, nil
}

func (s *Store) ListTemplates(ctx context.Context, tenantID string) ([]TemplateRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, body, created_at, updated_at
		FROM templates
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("store: failed to list templates: %w", err)
	}
	defer rows.Close()

	var records []TemplateRecord
	for rows.Next() {
		var r TemplateRecord
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Body, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store: template row scan error: %w", err)
		}
		records = append(records, r)
	}
	if records == nil {
		records = []TemplateRecord{}
	}
	return records, rows.Err()
}

func (s *Store) UpdateTemplate(ctx context.Context, tenantID, name, body string) (TemplateRecord, error) {
	var rec TemplateRecord
	err := s.pool.QueryRow(ctx, `
		UPDATE templates
		SET body = $3, updated_at = NOW()
		WHERE tenant_id = $1 AND name = $2
		RETURNING id, tenant_id, name, body, created_at, updated_at
	`, tenantID, name, body).Scan(
		&rec.ID, &rec.TenantID, &rec.Name, &rec.Body, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TemplateRecord{}, ErrTemplateNotFound
		}
		return TemplateRecord{}, fmt.Errorf("store: failed to update template: %w", err)
	}
	return rec, nil
}

func (s *Store) DeleteTemplate(ctx context.Context, tenantID, name string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM templates
		WHERE tenant_id = $1 AND name = $2
	`, tenantID, name)
	if err != nil {
		return fmt.Errorf("store: failed to delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "duplicate key")
}

func (s *Store) CreateTenant(ctx context.Context, name, apiKey string) (TenantRecord, error) {
	var rec TenantRecord
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, api_key)
		VALUES ($1, $2)
		RETURNING id, name, api_key, plan,
		          COALESCE(webhook_url, ''), COALESCE(webhook_secret, ''), created_at
	`, name, apiKey).Scan(
		&rec.ID, &rec.Name, &rec.APIKey, &rec.Plan,
		&rec.WebhookURL, &rec.WebhookSecret, &rec.CreatedAt,
	)
	if err != nil {
		return TenantRecord{}, fmt.Errorf("store: failed to create tenant: %w", err)
	}
	return rec, nil
}

func (s *Store) GetTenantByAPIKey(ctx context.Context, apiKey string) (TenantRecord, error) {
	var rec TenantRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, plan,
		       COALESCE(webhook_url, ''), COALESCE(webhook_secret, ''), created_at
		FROM tenants
		WHERE api_key = $1
	`, apiKey).Scan(
		&rec.ID, &rec.Name, &rec.APIKey, &rec.Plan,
		&rec.WebhookURL, &rec.WebhookSecret, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TenantRecord{}, fmt.Errorf("store: tenant not found: %w", err)
		}
		return TenantRecord{}, fmt.Errorf("store: failed to lookup tenant: %w", err)
	}
	return rec, nil
}

func (s *Store) GetTenantByID(ctx context.Context, tenantID string) (TenantRecord, error) {
	var rec TenantRecord
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, plan,
		       COALESCE(webhook_url, ''), COALESCE(webhook_secret, ''), created_at
		FROM tenants
		WHERE id = $1
	`, tenantID).Scan(
		&rec.ID, &rec.Name, &rec.APIKey, &rec.Plan,
		&rec.WebhookURL, &rec.WebhookSecret, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TenantRecord{}, ErrTenantNotFound
		}
		return TenantRecord{}, fmt.Errorf("store: failed to get tenant by id: %w", err)
	}
	return rec, nil
}

func (s *Store) SetTenantWebhook(ctx context.Context, tenantID, webhookURL, webhookSecret string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants
		SET webhook_url = $2, webhook_secret = $3
		WHERE id = $1
	`, tenantID, webhookURL, webhookSecret)
	if err != nil {
		return fmt.Errorf("store: failed to set tenant webhook: %w", err)
	}
	return nil
}

func (s *Store) ClearTenantWebhook(ctx context.Context, tenantID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants
		SET webhook_url = NULL, webhook_secret = NULL
		WHERE id = $1
	`, tenantID)
	if err != nil {
		return fmt.Errorf("store: failed to clear tenant webhook: %w", err)
	}
	return nil
}

func (s *Store) LogWebhookDelivery(ctx context.Context, rec WebhookDeliveryRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_deliveries
			(tenant_id, notification_request_id, status_code, attempt, success)
		VALUES ($1, $2, $3, $4, $5)
	`, rec.TenantID, rec.NotificationRequestID, rec.StatusCode, rec.Attempt, rec.Success)
	if err != nil {
		return fmt.Errorf("store: failed to log webhook delivery: %w", err)
	}
	return nil
}

func (s *Store) GetWebhookDeliveries(ctx context.Context, tenantID string, limit int) ([]WebhookDeliveryRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, notification_request_id, status_code, attempt, success, created_at
		FROM webhook_deliveries
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("store: failed to get webhook deliveries: %w", err)
	}
	defer rows.Close()

	var records []WebhookDeliveryRecord
	for rows.Next() {
		var r WebhookDeliveryRecord
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.NotificationRequestID,
			&r.StatusCode, &r.Attempt, &r.Success, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: webhook delivery scan error: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) LogNotification(ctx context.Context, rec NotificationRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications
			(tenant_id, request_id, recipient, channel, message, status,
			 error_message, attempts, fallback_used, idempotency_key, was_cached_response,
			 discord_recipient, email_recipient)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, rec.TenantID, rec.RequestID, rec.Recipient, rec.Channel, rec.Message, rec.Status,
		rec.ErrorMessage, rec.Attempts, rec.FallbackUsed, rec.IdempotencyKey, rec.WasCachedResponse,
		rec.DiscordRecipient, rec.EmailRecipient)
	if err != nil {
		return fmt.Errorf("store: failed to log notification: %w", err)
	}
	return nil
}

func (s *Store) CreateScheduled(ctx context.Context, rec NotificationRecord) (NotificationRecord, error) {
	var out NotificationRecord
	var sf *time.Time
	err := s.pool.QueryRow(ctx, `
		INSERT INTO notifications
			(tenant_id, request_id, recipient, channel, message, status,
			 error_message, attempts, fallback_used, idempotency_key, was_cached_response,
			 scheduled_for, discord_recipient, email_recipient)
		VALUES ($1, $2, $3, $4, $5, 'scheduled',
			 '', 0, false, $6, false,
			 $7, $8, $9)
		RETURNING id, tenant_id, request_id, recipient, channel, message, status,
				  error_message, attempts, fallback_used, idempotency_key,
				  was_cached_response, created_at, scheduled_for, discord_recipient, email_recipient
	`, rec.TenantID, rec.RequestID, rec.Recipient, rec.Channel, rec.Message,
		rec.IdempotencyKey, rec.ScheduledFor,
		rec.DiscordRecipient, rec.EmailRecipient,
	).Scan(
		&out.ID, &out.TenantID, &out.RequestID, &out.Recipient, &out.Channel, &out.Message, &out.Status,
		&out.ErrorMessage, &out.Attempts, &out.FallbackUsed, &out.IdempotencyKey,
		&out.WasCachedResponse, &out.CreatedAt, &sf, &out.DiscordRecipient, &out.EmailRecipient,
	)
	if err != nil {
		return NotificationRecord{}, fmt.Errorf("store: failed to create scheduled notification: %w", err)
	}
	out.ScheduledFor = sf
	return out, nil
}

func (s *Store) ClaimDueNotifications(ctx context.Context, limit int) ([]NotificationRecord, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE notifications
		SET status = 'processing'
		WHERE id IN (
			SELECT id FROM notifications
			WHERE status = 'scheduled' AND scheduled_for <= NOW()
			ORDER BY scheduled_for ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, tenant_id, request_id, recipient, channel, message, status,
				  error_message, attempts, fallback_used, idempotency_key,
				  was_cached_response, created_at, scheduled_for, discord_recipient, email_recipient
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("store: failed to claim due notifications: %w", err)
	}
	defer rows.Close()

	var records []NotificationRecord
	for rows.Next() {
		var r NotificationRecord
		var sf *time.Time
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.RequestID, &r.Recipient, &r.Channel, &r.Message, &r.Status,
			&r.ErrorMessage, &r.Attempts, &r.FallbackUsed, &r.IdempotencyKey,
			&r.WasCachedResponse, &r.CreatedAt, &sf, &r.DiscordRecipient, &r.EmailRecipient,
		); err != nil {
			return nil, fmt.Errorf("store: claim scan error: %w", err)
		}
		r.ScheduledFor = sf
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) UpdateNotificationStatus(
	ctx context.Context,
	requestID, status, errMsg string,
	attempts int,
	fallbackUsed bool,
) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		SET status = $2, error_message = $3, attempts = $4, fallback_used = $5
		WHERE request_id = $1
	`, requestID, status, errMsg, attempts, fallbackUsed)
	if err != nil {
		return fmt.Errorf("store: failed to update notification status: %w", err)
	}
	return nil
}

func (s *Store) GetNotificationByRequestID(ctx context.Context, tenantID, requestID string) (NotificationRecord, error) {
	var r NotificationRecord
	var sf *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, request_id, recipient, channel, message, status,
			   error_message, attempts, fallback_used, idempotency_key,
			   was_cached_response, created_at, scheduled_for, discord_recipient, email_recipient
		FROM notifications
		WHERE request_id = $1 AND tenant_id = $2
	`, requestID, tenantID).Scan(
		&r.ID, &r.TenantID, &r.RequestID, &r.Recipient, &r.Channel, &r.Message, &r.Status,
		&r.ErrorMessage, &r.Attempts, &r.FallbackUsed, &r.IdempotencyKey,
		&r.WasCachedResponse, &r.CreatedAt, &sf, &r.DiscordRecipient, &r.EmailRecipient,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return NotificationRecord{}, ErrNotificationNotFound
		}
		return NotificationRecord{}, fmt.Errorf("store: failed to get notification: %w", err)
	}
	r.ScheduledFor = sf
	return r, nil
}

func (s *Store) CancelScheduledNotification(ctx context.Context, tenantID, requestID string) error {
	r, err := s.GetNotificationByRequestID(ctx, tenantID, requestID)
	if err != nil {
		return err
	}
	if r.Status != "scheduled" {
		return ErrNotificationAlreadySent
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE notifications
		SET status = 'cancelled'
		WHERE request_id = $1 AND tenant_id = $2 AND status = 'scheduled'
	`, requestID, tenantID)
	if err != nil {
		return fmt.Errorf("store: failed to cancel notification: %w", err)
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
		       was_cached_response, created_at, scheduled_for, discord_recipient, email_recipient
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
		var sf *time.Time
		if err := rows.Scan(
			&r.ID, &r.TenantID, &r.RequestID, &r.Recipient, &r.Channel,
			&r.Message, &r.Status, &r.ErrorMessage, &r.Attempts,
			&r.FallbackUsed, &r.IdempotencyKey, &r.WasCachedResponse, &r.CreatedAt,
			&sf, &r.DiscordRecipient, &r.EmailRecipient,
		); err != nil {
			return nil, fmt.Errorf("store: row scan error: %w", err)
		}
		r.ScheduledFor = sf
		records = append(records, r)
	}

	return records, rows.Err()
}

type TenantUsage struct {
	Count    int
	OldestAt *time.Time
}

func (s *Store) GetTenantUsage(ctx context.Context, tenantID string) (TenantUsage, error) {
	if tenantID == "" {
		return TenantUsage{}, nil
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	var usage TenantUsage
	var oldest *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*), MIN(created_at)
		FROM notifications
		WHERE tenant_id = $1 AND created_at >= $2
	`, tenantID, cutoff).Scan(&usage.Count, &oldest)
	if err != nil {
		return TenantUsage{}, fmt.Errorf("store: failed to get tenant usage: %w", err)
	}
	usage.OldestAt = oldest
	return usage, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

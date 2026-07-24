package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"relayhub/internal/store"
)

func Sign(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func Verify(secret, body []byte, sig string) bool {
	expected := Sign(secret, body)
	return hmac.Equal([]byte(expected), []byte(sig))
}

type EventPayload struct {
	Event        string    `json:"event"`
	RequestID    string    `json:"request_id"`
	ChannelUsed  string    `json:"channel_used"`
	FallbackUsed bool      `json:"fallback_used"`
	Attempts     int       `json:"attempts"`
	Timestamp    time.Time `json:"timestamp"`
}

type RetryFunc func(fn func() error, maxAttempts int, logger *slog.Logger) (int, error)

type Dispatcher struct {
	store       *store.Store
	httpClient  *http.Client
	logger      *slog.Logger
	retry       RetryFunc
	maxAttempts int
}

type Params struct {
	Store       *store.Store
	Retry       RetryFunc
	MaxAttempts int
	Logger      *slog.Logger
}

func New(p Params) *Dispatcher {
	maxAttempts := p.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &Dispatcher{
		store: p.Store,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:      p.Logger,
		retry:       p.Retry,
		maxAttempts: maxAttempts,
	}
}

func NewWithClient(p Params, client *http.Client) *Dispatcher {
	d := New(p)
	d.httpClient = client
	return d
}

func (d *Dispatcher) FireAsync(tenantID, webhookURL, webhookSecret string, payload EventPayload) {
	if webhookURL == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		d.deliver(ctx, tenantID, webhookURL, webhookSecret, payload)
	}()
}

func (d *Dispatcher) deliver(
	ctx context.Context,
	tenantID, webhookURL, webhookSecret string,
	payload EventPayload,
) {
	log := d.logger.With(
		"tenant_id", tenantID,
		"request_id", payload.RequestID,
		"webhook_url", webhookURL,
	)

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error("webhook: failed to marshal payload", "error", err)
		return
	}

	secretBytes, err := hex.DecodeString(webhookSecret)
	if err != nil {
		secretBytes = []byte(webhookSecret)
	}
	sig := Sign(secretBytes, body)

	attempt := 0
	_, _ = d.retry(func() error {
		attempt++
		statusCode, success, sendErr := d.post(ctx, webhookURL, sig, body)

		logCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if logErr := d.store.LogWebhookDelivery(logCtx, store.WebhookDeliveryRecord{
			TenantID:              tenantID,
			NotificationRequestID: payload.RequestID,
			StatusCode:            statusCode,
			Attempt:               attempt,
			Success:               success,
		}); logErr != nil {
			log.Warn("webhook: failed to log delivery attempt", "error", logErr)
		}

		if success {
			log.Info("webhook: delivered successfully", "attempt", attempt, "status", statusCode)
		}
		return sendErr
	}, d.maxAttempts, log)
}

func (d *Dispatcher) post(ctx context.Context, url, sig string, body []byte) (int, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, false, fmt.Errorf("webhook: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RelayHub-Signature", sig)
	req.Header.Set("User-Agent", "RelayHub-Webhooks/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("webhook: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, false,
			fmt.Errorf("webhook: non-2xx response from endpoint: %d", resp.StatusCode)
	}
	return resp.StatusCode, true, nil
}

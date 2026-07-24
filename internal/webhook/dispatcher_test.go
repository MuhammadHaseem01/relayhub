package webhook_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"relayhub/internal/store"
	"relayhub/internal/webhook"
)

func openTestDB(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	db, err := store.New(url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func uniqueKey(prefix string) string {
	return fmt.Sprintf("rh_wh_%s_%d", prefix, time.Now().UnixNano())
}

func noopRetry(fn func() error, maxAttempts int, _ *slog.Logger) (int, error) {
	var err error
	for i := 1; i <= maxAttempts; i++ {
		err = fn()
		if err == nil {
			return i, nil
		}
	}
	return maxAttempts, err
}

func TestDispatcher_SuccessDelivery(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()

	tenant, err := db.CreateTenant(ctx, "WH Success Tenant", uniqueKey("succ"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	secret := "testsecret1234567890abcdef123456"
	if err := db.SetTenantWebhook(ctx, tenant.ID, "https://example.com", secret); err != nil {
		t.Fatalf("SetTenantWebhook: %v", err)
	}

	received := make(chan []byte, 1)
	receivedSig := make(chan string, 1)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		receivedSig <- r.Header.Get("X-RelayHub-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	d := webhook.NewWithClient(webhook.Params{
		Store:       db,
		Retry:       noopRetry,
		MaxAttempts: 3,
		Logger:      slog.Default(),
	}, srv.Client())

	payload := webhook.EventPayload{
		Event:        "notification.delivered",
		RequestID:    "req-disp-success-001",
		ChannelUsed:  "email",
		FallbackUsed: false,
		Attempts:     1,
		Timestamp:    time.Now().UTC(),
	}

	d.FireAsync(tenant.ID, srv.URL, secret, payload)

	select {
	case body := <-received:
		sig := <-receivedSig

		secretBytes := []byte(secret)
		if !webhook.Verify(secretBytes, body, sig) {
			t.Errorf("signature verification failed: sig=%s", sig)
		}
		var got webhook.EventPayload
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("body not valid JSON: %v", err)
		}
		if got.RequestID != payload.RequestID {
			t.Errorf("request_id mismatch: got %q want %q", got.RequestID, payload.RequestID)
		}
		if got.Event != payload.Event {
			t.Errorf("event mismatch: got %q want %q", got.Event, payload.Event)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}

	time.Sleep(200 * time.Millisecond)
	deliveries, err := db.GetWebhookDeliveries(ctx, tenant.ID, 10)
	if err != nil {
		t.Fatalf("GetWebhookDeliveries: %v", err)
	}
	if len(deliveries) == 0 {
		t.Fatal("expected at least one delivery log row, got 0")
	}
	found := false
	for _, d := range deliveries {
		if d.NotificationRequestID == payload.RequestID && d.Success {
			found = true
			break
		}
	}
	if !found {
		t.Error("no successful delivery row found for the request_id")
	}
}

func TestDispatcher_RetryOn500(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()

	tenant, err := db.CreateTenant(ctx, "WH Retry Tenant", uniqueKey("retry"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	var mu sync.Mutex
	callCount := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	const maxAttempts = 3
	d := webhook.NewWithClient(webhook.Params{
		Store:       db,
		Retry:       noopRetry,
		MaxAttempts: maxAttempts,
		Logger:      slog.Default(),
	}, srv.Client())

	requestID := "req-retry-500-001"
	payload := webhook.EventPayload{
		Event:       "notification.failed",
		RequestID:   requestID,
		ChannelUsed: "email",
		Timestamp:   time.Now().UTC(),
	}

	d.FireAsync(tenant.ID, srv.URL, "anysecret", payload)

	time.Sleep(1 * time.Second)

	mu.Lock()
	got := callCount
	mu.Unlock()

	if got != maxAttempts {
		t.Errorf("expected %d HTTP calls (retries), got %d", maxAttempts, got)
	}

	deliveries, err := db.GetWebhookDeliveries(ctx, tenant.ID, 20)
	if err != nil {
		t.Fatalf("GetWebhookDeliveries: %v", err)
	}

	var matching int
	for _, d := range deliveries {
		if d.NotificationRequestID == requestID {
			matching++
			if d.Success {
				t.Errorf("attempt %d logged as success=true but server returned 500", d.Attempt)
			}
			if d.StatusCode != http.StatusInternalServerError {
				t.Errorf("expected status_code=500, got %d", d.StatusCode)
			}
		}
	}
	if matching != maxAttempts {
		t.Errorf("expected %d delivery log rows, got %d", maxAttempts, matching)
	}
}

func TestDispatcher_NoWebhookURL(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()

	tenant, err := db.CreateTenant(ctx, "WH NoURL Tenant", uniqueKey("nourl"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	httpCalled := false
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	d := webhook.NewWithClient(webhook.Params{
		Store:       db,
		Retry:       noopRetry,
		MaxAttempts: 3,
		Logger:      slog.Default(),
	}, srv.Client())

	d.FireAsync(tenant.ID, "", "anysecret", webhook.EventPayload{
		Event:     "notification.delivered",
		RequestID: "req-no-url-001",
		Timestamp: time.Now().UTC(),
	})

	time.Sleep(300 * time.Millisecond)

	if httpCalled {
		t.Error("HTTP call was made even though webhookURL was empty")
	}

	deliveries, err := db.GetWebhookDeliveries(ctx, tenant.ID, 10)
	if err != nil {
		t.Fatalf("GetWebhookDeliveries: %v", err)
	}
	if len(deliveries) != 0 {
		t.Errorf("expected 0 delivery rows for no-op, got %d", len(deliveries))
	}
}

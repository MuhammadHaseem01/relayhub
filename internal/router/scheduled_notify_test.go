package router_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"relayhub/internal/store"
)

func ctxWithTimeout(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestNotify_SendAt_Future_Returns202(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	sendAt := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "future@example.com",
		"message":   "scheduled hello",
		"send_at":   sendAt,
	})
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["status"] != "scheduled" {
		t.Errorf("expected status=scheduled, got %v", data["status"])
	}
	if data["request_id"] == "" {
		t.Error("expected non-empty request_id")
	}
	if data["scheduled_for"] == "" {
		t.Error("expected non-empty scheduled_for")
	}

	if svc.lastReq.Message != "" {
		t.Error("expected no send call for scheduled notification")
	}
}

func TestNotify_SendAt_Past_SendsImmediately(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	pastTime := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "past@example.com",
		"message":   "should send now",
		"send_at":   pastTime,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if svc.lastReq.Message != "should send now" {
		t.Errorf("message not forwarded: %q", svc.lastReq.Message)
	}
}

func TestNotify_SendAt_Omitted_SendsImmediately(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "now@example.com",
		"message":   "immediate",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotify_SendAt_TooFarFuture_Returns400(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	tooFar := time.Now().Add(31 * 24 * time.Hour).UTC().Format(time.RFC3339)
	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "far@example.com",
		"message":   "too far",
		"send_at":   tooFar,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotify_SendAt_InvalidFormat_Returns400(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "x@example.com",
		"message":   "bad date",
		"send_at":   "not-a-date",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestNotify_WithTemplateAndSendAt_MessageRenderedAtRequestTime(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	createTemplate(t, db, tenantID, "sched_tmpl", "Hello {{name}}!")

	sendAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "x@example.com",
		"template":  "sched_tmpl",
		"variables": map[string]string{"name": "World"},
		"send_at":   sendAt,
	})
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	reqID, _ := data["request_id"].(string)

	stored, err := db.GetNotificationByRequestID(ctxWithTimeout(t), tenantID, reqID)
	if err != nil {
		t.Fatalf("GetNotificationByRequestID: %v", err)
	}
	if stored.Message != "Hello World!" {
		t.Errorf("message not rendered at request time: got %q", stored.Message)
	}
}

func TestGetNotification_OK(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	futureTime := time.Now().Add(2 * time.Hour)
	rec, err := db.CreateScheduled(ctxWithTimeout(t), store.NotificationRecord{
		TenantID:     tenantID,
		RequestID:    "get-notif-req-" + uniqueRouterName("g"),
		Recipient:    "x@example.com",
		Channel:      "email",
		Message:      "hello",
		ScheduledFor: &futureTime,
	})
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}

	w := doRequest(t, h, "GET", "/v1/notify/"+rec.RequestID, key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["request_id"] != rec.RequestID {
		t.Errorf("request_id mismatch: %v", data["request_id"])
	}
	if data["status"] != "scheduled" {
		t.Errorf("expected status=scheduled, got %v", data["status"])
	}
}

func TestGetNotification_TenantIsolation_Returns404(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantAID, _ := createTenantAndKey(t, db)
	_, keyB := createTenantAndKey(t, db)

	futureTime := time.Now().Add(2 * time.Hour)
	rec, _ := db.CreateScheduled(ctxWithTimeout(t), store.NotificationRecord{
		TenantID:     tenantAID,
		RequestID:    "iso-get-" + uniqueRouterName("ig"),
		Recipient:    "x@example.com",
		Channel:      "email",
		Message:      "private",
		ScheduledFor: &futureTime,
	})

	w := doRequest(t, h, "GET", "/v1/notify/"+rec.RequestID, keyB, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (tenant isolation), got %d", w.Code)
	}
}

func TestGetNotification_NotFound_Returns404(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "GET", "/v1/notify/totally-fake-id", key, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCancelNotification_OK_Returns204(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	futureTime := time.Now().Add(2 * time.Hour)
	rec, _ := db.CreateScheduled(ctxWithTimeout(t), store.NotificationRecord{
		TenantID:     tenantID,
		RequestID:    "cancel-ok-" + uniqueRouterName("co"),
		Recipient:    "x@example.com",
		Channel:      "email",
		Message:      "will be cancelled",
		ScheduledFor: &futureTime,
	})

	w := doRequest(t, h, "DELETE", "/v1/notify/"+rec.RequestID, key, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	got, _ := db.GetNotificationByRequestID(ctxWithTimeout(t), tenantID, rec.RequestID)
	if got.Status != "cancelled" {
		t.Errorf("expected status=cancelled after delete, got %q", got.Status)
	}
}

func TestCancelNotification_AlreadySent_Returns409(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	pastTime := time.Now().Add(-1 * time.Minute)
	rec, _ := db.CreateScheduled(ctxWithTimeout(t), store.NotificationRecord{
		TenantID:     tenantID,
		RequestID:    "cancel-sent-" + uniqueRouterName("cs"),
		Recipient:    "x@example.com",
		Channel:      "email",
		Message:      "already processing",
		ScheduledFor: &pastTime,
	})

	_, _ = db.ClaimDueNotifications(ctxWithTimeout(t), 10)

	w := doRequest(t, h, "DELETE", "/v1/notify/"+rec.RequestID, key, nil)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCancelNotification_NotFound_Returns404(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "DELETE", "/v1/notify/no-such-id", key, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCancelNotification_TenantIsolation_Returns404(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantAID, _ := createTenantAndKey(t, db)
	_, keyB := createTenantAndKey(t, db)

	futureTime := time.Now().Add(2 * time.Hour)
	rec, _ := db.CreateScheduled(ctxWithTimeout(t), store.NotificationRecord{
		TenantID:     tenantAID,
		RequestID:    "iso-cancel-" + uniqueRouterName("ic"),
		Recipient:    "x@example.com",
		Channel:      "email",
		Message:      "private",
		ScheduledFor: &futureTime,
	})

	w := doRequest(t, h, "DELETE", "/v1/notify/"+rec.RequestID, keyB, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (tenant isolation), got %d", w.Code)
	}
}

func TestScheduleEndpoints_RequireAuth(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)

	endpoints := []struct{ method, path string }{
		{"GET", "/v1/notify/some-id"},
		{"DELETE", "/v1/notify/some-id"},
	}
	for _, e := range endpoints {
		t.Run(e.method+" "+e.path, func(t *testing.T) {
			w := doRequest(t, h, e.method, e.path, "", nil)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

package router_test

import (
	"context"
	"net/http"
	"testing"

	"relayhub/internal/health"
	"relayhub/internal/queue"
	"relayhub/internal/store"
)

func TestGetProviderHealth_PublicEndpoint_OK(t *testing.T) {
	db := openRouterDB(t)
	h := newTestServer(t, db)

	w := doRequest(t, h, "GET", "/v1/health/providers", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["discord"] != "healthy" || data["email"] != "healthy" || data["smtp"] != "healthy" {
		t.Errorf("unexpected health data: %v", data)
	}
}

func TestGetDeadLetter_OK(t *testing.T) {
	db := openRouterDB(t)
	h := newTestServer(t, db)
	tenantID, key := createTenantAndKey(t, db)

	ctx := context.Background()
	reqID := "dlq-req-" + uniqueRouterName("dl")
	_, err := db.CreateQueued(ctx, store.NotificationRecord{
		TenantID:  tenantID,
		RequestID: reqID,
		Recipient: "dead@example.com",
		Channel:   "email",
		Message:   "dead letter test",
	})
	if err != nil {
		t.Fatalf("CreateQueued: %v", err)
	}

	if err := db.UpdateNotificationStatus(ctx, reqID, "dead_letter", "exceeded worker attempts", 3, false); err != nil {
		t.Fatalf("UpdateNotificationStatus: %v", err)
	}

	w := doRequest(t, h, "GET", "/v1/notify/dead-letter", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v1/notify/dead-letter: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	count, _ := data["count"].(float64)
	if count != 1 {
		t.Errorf("expected count=1, got %v", count)
	}
}

func TestGetDeadLetter_TenantIsolation(t *testing.T) {
	db := openRouterDB(t)
	h := newTestServer(t, db)

	tenantAID, _ := createTenantAndKey(t, db)
	_, keyB := createTenantAndKey(t, db)

	ctx := context.Background()
	reqIDA := "iso-dlq-" + uniqueRouterName("a")
	_, _ = db.CreateQueued(ctx, store.NotificationRecord{
		TenantID:  tenantAID,
		RequestID: reqIDA,
		Recipient: "a@example.com",
		Channel:   "email",
		Message:   "tenant A DLQ",
	})
	_ = db.UpdateNotificationStatus(ctx, reqIDA, "dead_letter", "failed", 3, false)

	w := doRequest(t, h, "GET", "/v1/notify/dead-letter", keyB, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	count, _ := data["count"].(float64)
	if count != 0 {
		t.Errorf("tenant B should see 0 dead letter items, got %v", count)
	}
}

func TestReplayNotification_OK(t *testing.T) {
	db := openRouterDB(t)
	h := newTestServer(t, db)
	tenantID, key := createTenantAndKey(t, db)

	ctx := context.Background()
	reqID := "replay-req-" + uniqueRouterName("r")
	_, err := db.CreateQueued(ctx, store.NotificationRecord{
		TenantID:  tenantID,
		RequestID: reqID,
		Recipient: "replay@example.com",
		Channel:   "email",
		Message:   "replay test",
	})
	if err != nil {
		t.Fatalf("CreateQueued: %v", err)
	}
	_ = db.UpdateNotificationStatus(ctx, reqID, "dead_letter", "failed 3 times", 3, false)

	w := doRequest(t, h, "POST", "/v1/notify/"+reqID+"/replay", key, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /v1/notify/%s/replay: expected 200, got %d: %s", reqID, w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["status"] != "queued" {
		t.Errorf("expected status=queued after replay, got %v", data["status"])
	}

	rec, err := db.GetNotificationByRequestID(ctx, tenantID, reqID)
	if err != nil {
		t.Fatalf("GetNotificationByRequestID: %v", err)
	}
	if rec.Status != "queued" {
		t.Errorf("expected DB status=queued, got %q", rec.Status)
	}
}

func TestReplayNotification_NotFound_Returns404(t *testing.T) {
	db := openRouterDB(t)
	h := newTestServer(t, db)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/notify/fake-req-id/replay", key, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReplayNotification_NotDeadLetter_Returns409(t *testing.T) {
	db := openRouterDB(t)
	h := newTestServer(t, db)
	tenantID, key := createTenantAndKey(t, db)

	ctx := context.Background()
	reqID := "delivered-req-" + uniqueRouterName("d")
	_, err := db.CreateQueued(ctx, store.NotificationRecord{
		TenantID:  tenantID,
		RequestID: reqID,
		Recipient: "delivered@example.com",
		Channel:   "email",
		Message:   "delivered test",
	})
	if err != nil {
		t.Fatalf("CreateQueued: %v", err)
	}
	_ = db.UpdateNotificationStatus(ctx, reqID, "delivered", "", 1, false)

	w := doRequest(t, h, "POST", "/v1/notify/"+reqID+"/replay", key, nil)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for non-dead_letter replay, got %d: %s", w.Code, w.Body.String())
	}
}

var _ = health.NewRegistry
var _ = queue.NewFromClient

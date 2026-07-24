package router_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func uniqueWebhookName(prefix string) string {
	return fmt.Sprintf("rh_wh_%s_%d", prefix, time.Now().UnixNano())
}

func TestSetWebhook_OK(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	handler := newTestServer(db, svc)

	_, apiKey := createTenantAndKey(t, db)

	w := doRequest(t, handler, http.MethodPut, "/v1/webhook", apiKey, map[string]string{
		"webhook_url": "https://example.com/events",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /v1/webhook: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	data := decodeResponse(t, w)
	d, _ := data["data"].(map[string]any)
	secret, _ := d["webhook_secret"].(string)
	if secret == "" {
		t.Fatal("expected non-empty webhook_secret in response")
	}
	if url, _ := d["webhook_url"].(string); url != "https://example.com/events" {
		t.Errorf("webhook_url mismatch: got %q", url)
	}

	w2 := doRequest(t, handler, http.MethodPut, "/v1/webhook", apiKey, map[string]string{
		"webhook_url": "https://example.com/events-v2",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("second PUT /v1/webhook: expected 200, got %d", w2.Code)
	}
	data2 := decodeResponse(t, w2)
	d2, _ := data2["data"].(map[string]any)
	secret2, _ := d2["webhook_secret"].(string)
	if secret2 != secret {
		t.Errorf("expected secret to be reused on update, got different secret")
	}
}

func TestSetWebhook_RequiresHTTPS(t *testing.T) {
	db := openRouterDB(t)
	handler := newTestServer(db, &stubNotifyService{})
	_, apiKey := createTenantAndKey(t, db)

	w := doRequest(t, handler, http.MethodPut, "/v1/webhook", apiKey, map[string]string{
		"webhook_url": "http://insecure.example.com/events",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for http:// URL, got %d", w.Code)
	}
}

func TestDeleteWebhook_OK(t *testing.T) {
	db := openRouterDB(t)
	handler := newTestServer(db, &stubNotifyService{})
	tenantID, apiKey := createTenantAndKey(t, db)

	doRequest(t, handler, http.MethodPut, "/v1/webhook", apiKey, map[string]string{
		"webhook_url": "https://example.com/events",
	})
	w := doRequest(t, handler, http.MethodDelete, "/v1/webhook", apiKey, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE /v1/webhook: expected 204, got %d — body: %s", w.Code, w.Body.String())
	}

	ctx := context.Background()
	tenant, err := db.GetTenantByID(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetTenantByID: %v", err)
	}
	if tenant.WebhookURL != "" {
		t.Errorf("expected webhook_url to be empty after DELETE, got %q", tenant.WebhookURL)
	}
}

func TestGetWebhookDeliveries_OK(t *testing.T) {
	db := openRouterDB(t)
	handler := newTestServer(db, &stubNotifyService{})
	_, apiKey := createTenantAndKey(t, db)

	w := doRequest(t, handler, http.MethodGet, "/v1/webhook/deliveries", apiKey, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v1/webhook/deliveries: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	data := decodeResponse(t, w)
	d, _ := data["data"].(map[string]any)
	if _, ok := d["count"]; !ok {
		t.Error("response missing 'count' field")
	}
	if _, ok := d["deliveries"]; !ok {
		t.Error("response missing 'deliveries' field")
	}
}

func TestSetWebhook_Unauthorized(t *testing.T) {
	db := openRouterDB(t)
	handler := newTestServer(db, &stubNotifyService{})

	w := doRequest(t, handler, http.MethodPut, "/v1/webhook", "", map[string]string{
		"webhook_url": "https://example.com/events",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

var _ = uniqueWebhookName
var _ = slog.Default
var _ = fmt.Sprintf

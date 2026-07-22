package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"log/slog"

	"relayhub/internal/middleware"
	"relayhub/internal/router"
	"relayhub/internal/service/notify_service"
	"relayhub/internal/store"
)

type stubNotifyService struct {
	lastReq notify_service.Request
}

func (s *stubNotifyService) Send(_ context.Context, req notify_service.Request) (notify_service.Response, error) {
	s.lastReq = req
	return notify_service.Response{
		RequestID: "stub-req-id",
		Status:    "delivered",
		Channel:   req.Channel,
	}, nil
}

func openRouterDB(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping router integration test")
	}
	db, err := store.New(url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func newTestServer(db *store.Store, svc notify_service.NotifyService) http.Handler {
	return router.New(router.Config{
		NotifyService: svc,
		Store:         db,
		Logger:        slog.Default(),
	})
}

func doRequest(t *testing.T, handler http.Handler, method, path, apiKey string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("json.Encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return m
}

func uniqueRouterName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func createTenantAndKey(t *testing.T, db *store.Store) (string, string) {
	t.Helper()
	ctx := context.Background()
	apiKey := "rh_router_tmpl_" + uniqueRouterName("k")
	tenant, err := db.CreateTenant(ctx, "Router Test Tenant "+apiKey, apiKey)
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	return tenant.ID, apiKey
}

func createTemplate(t *testing.T, db *store.Store, tenantID, name, body string) store.TemplateRecord {
	t.Helper()
	ctx := context.Background()
	tmpl, err := db.CreateTemplate(ctx, tenantID, name, body)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	return tmpl
}

func TestHandleCreateTemplate_OK(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	body := map[string]string{
		"name": "welcome_email",
		"body": "Hi {{customer_name}}, welcome!",
	}
	w := doRequest(t, h, "POST", "/v1/templates", key, body)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["name"] != "welcome_email" {
		t.Errorf("name mismatch: %v", data["name"])
	}
}

func TestHandleCreateTemplate_InvalidName(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/templates", key, map[string]string{
		"name": "bad name!",
		"body": "body",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCreateTemplate_BodyTooLong(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	longBody := make([]byte, 4001)
	for i := range longBody {
		longBody[i] = 'x'
	}
	w := doRequest(t, h, "POST", "/v1/templates", key, map[string]string{
		"name": "long_body",
		"body": string(longBody),
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCreateTemplate_Duplicate_Returns409(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	payload := map[string]string{"name": "duplicate_tmpl", "body": "hello"}
	doRequest(t, h, "POST", "/v1/templates", key, payload)      // first → 201
	w := doRequest(t, h, "POST", "/v1/templates", key, payload) // second → 409
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleListTemplates_TenantIsolation(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)

	tenantAID, keyA := createTenantAndKey(t, db)
	_, keyB := createTenantAndKey(t, db)

	createTemplate(t, db, tenantAID, "tmpl_for_a", "body")

	w := doRequest(t, h, "GET", "/v1/templates", keyB, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	count, _ := data["count"].(float64)
	if count != 0 {
		t.Errorf("tenant B should see 0 templates, got %v", count)
	}

	wA := doRequest(t, h, "GET", "/v1/templates", keyA, nil)
	respA := decodeResponse(t, wA)
	dataA, _ := respA["data"].(map[string]any)
	countA, _ := dataA["count"].(float64)
	if countA != 1 {
		t.Errorf("tenant A should see 1 template, got %v", countA)
	}
}

func TestHandleGetTemplate_NotFound(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "GET", "/v1/templates/ghost", key, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleUpdateTemplate_OK(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	createTemplate(t, db, tenantID, "editable", "old body")

	w := doRequest(t, h, "PUT", "/v1/templates/editable", key, map[string]string{
		"body": "new body {{x}}",
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["body"] != "new body {{x}}" {
		t.Errorf("body not updated: %v", data["body"])
	}
}

func TestHandleDeleteTemplate_OK(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	createTemplate(t, db, tenantID, "gone", "bye")

	w := doRequest(t, h, "DELETE", "/v1/templates/gone", key, nil)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	wGet := doRequest(t, h, "GET", "/v1/templates/gone", key, nil)
	if wGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", wGet.Code)
	}
}

func TestHandleDeleteTemplate_NotFound(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "DELETE", "/v1/templates/no_such_thing", key, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNotify_PlainMessage_StillWorks(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "a@example.com",
		"message":   "Hello plain world!",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if svc.lastReq.Message != "Hello plain world!" {
		t.Errorf("message not passed through: %q", svc.lastReq.Message)
	}
}

func TestNotify_WithTemplate_OK(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	createTemplate(t, db, tenantID, "order_shipped",
		"Hi {{customer_name}}, your order {{order_id}} has shipped!")

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "ali@example.com",
		"template":  "order_shipped",
		"variables": map[string]string{
			"customer_name": "Ali",
			"order_id":      "4471",
		},
	})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	want := "Hi Ali, your order 4471 has shipped!"
	if svc.lastReq.Message != want {
		t.Errorf("substituted message wrong: got %q, want %q", svc.lastReq.Message, want)
	}
}

func TestNotify_WithTemplate_MissingVar_Returns400(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	createTemplate(t, db, tenantID, "needs_vars", "Hello {{name}}, code is {{code}}.")

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "x@example.com",
		"template":  "needs_vars",
		"variables": map[string]string{"name": "Bob"},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	missing, _ := resp["missing_variables"].([]any)
	if len(missing) == 0 {
		t.Errorf("expected missing_variables in response, got: %v", resp)
	}
	found := false
	for _, m := range missing {
		if m == "code" {
			found = true
		}
	}
	if !found {
		t.Errorf("'code' not listed in missing_variables: %v", missing)
	}
}

func TestNotify_WithTemplate_NotFound_Returns404(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "x@example.com",
		"template":  "ghost_template",
		"variables": map[string]string{},
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotify_BothMessageAndTemplate_Returns400(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	_, key := createTenantAndKey(t, db)

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "x@example.com",
		"message":   "direct message",
		"template":  "some_template",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotify_TenantCannotUseOtherTenantTemplate(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)

	tenantAID, _ := createTenantAndKey(t, db)
	_, keyB := createTenantAndKey(t, db)

	createTemplate(t, db, tenantAID, "private_tmpl", "Secret: {{secret}}")

	w := doRequest(t, h, "POST", "/v1/notify", keyB, map[string]any{
		"channel":   "email",
		"recipient": "b@example.com",
		"template":  "private_tmpl",
		"variables": map[string]string{"secret": "42"},
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (tenant isolation), got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotify_WithTemplate_NoVariables_NoPlaceholders(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)
	tenantID, key := createTenantAndKey(t, db)

	createTemplate(t, db, tenantID, "static_msg", "This is a static message.")

	w := doRequest(t, h, "POST", "/v1/notify", key, map[string]any{
		"channel":   "email",
		"recipient": "x@example.com",
		"template":  "static_msg",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTemplateEndpoints_RequireAuth(t *testing.T) {
	db := openRouterDB(t)
	svc := &stubNotifyService{}
	h := newTestServer(db, svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/v1/templates"},
		{"GET", "/v1/templates"},
		{"GET", "/v1/templates/any"},
		{"PUT", "/v1/templates/any"},
		{"DELETE", "/v1/templates/any"},
	}

	for _, e := range endpoints {
		t.Run(e.method+" "+e.path, func(t *testing.T) {
			var body any
			if e.method == "POST" || e.method == "PUT" {
				body = map[string]string{"name": "x", "body": "y"}
			}
			w := doRequest(t, h, e.method, e.path, "", body)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

var _ = middleware.WithTenant

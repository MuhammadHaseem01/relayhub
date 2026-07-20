package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"relayhub/internal/middleware"
	"relayhub/internal/store"
)

type mockTenantDB struct {
	records map[string]store.TenantRecord
}

func newMockDB(records ...store.TenantRecord) *mockTenantDB {
	m := &mockTenantDB{records: make(map[string]store.TenantRecord)}
	for _, r := range records {
		m.records[r.APIKey] = r
	}
	return m
}

func authMiddlewareFn(db *mockTenantDB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":"X-API-Key header is required"}`))
				return
			}

			tenant, ok := db.records[apiKey]
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":"invalid API key"}`))
				return
			}

			ctx := middleware.WithTenant(r.Context(), tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type sentinelHandler struct {
	called bool
	tenant store.TenantRecord
}

func (h *sentinelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	h.tenant, _ = middleware.TenantFromContext(r.Context())
	w.WriteHeader(http.StatusOK)
}

func TestAuth_MissingKey_Returns401(t *testing.T) {
	t.Parallel()

	db := newMockDB()
	sentinel := &sentinelHandler{}
	handler := authMiddlewareFn(db)(sentinel)

	req := httptest.NewRequest(http.MethodGet, "/v1/logs", nil)
	// No X-API-Key header.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if sentinel.called {
		t.Error("downstream handler must NOT be called when key is missing")
	}
	if !strings.Contains(w.Body.String(), "X-API-Key header is required") {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestAuth_InvalidKey_Returns401(t *testing.T) {
	t.Parallel()

	db := newMockDB(store.TenantRecord{
		ID:     "tenant-1",
		Name:   "Acme Corp",
		APIKey: "rh_valid_key_abc",
		Plan:   "free",
	})
	sentinel := &sentinelHandler{}
	handler := authMiddlewareFn(db)(sentinel)

	req := httptest.NewRequest(http.MethodGet, "/v1/logs", nil)
	req.Header.Set("X-API-Key", "rh_TOTALLY_WRONG")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if sentinel.called {
		t.Error("downstream handler must NOT be called for an invalid key")
	}
	if !strings.Contains(w.Body.String(), "invalid API key") {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestAuth_ValidKey_PassesThroughAndAttachesTenant(t *testing.T) {
	t.Parallel()

	expected := store.TenantRecord{
		ID:     "tenant-abc-123",
		Name:   "Test Corp",
		APIKey: "rh_correct_key_xyz",
		Plan:   "free",
	}
	db := newMockDB(expected)
	sentinel := &sentinelHandler{}
	handler := authMiddlewareFn(db)(sentinel)

	req := httptest.NewRequest(http.MethodPost, "/v1/notify", nil)
	req.Header.Set("X-API-Key", "rh_correct_key_xyz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !sentinel.called {
		t.Error("downstream handler MUST be called for a valid key")
	}
	if sentinel.tenant.ID != expected.ID {
		t.Errorf("tenant ID: got %q, want %q", sentinel.tenant.ID, expected.ID)
	}
	if sentinel.tenant.Name != expected.Name {
		t.Errorf("tenant name: got %q, want %q", sentinel.tenant.Name, expected.Name)
	}
}

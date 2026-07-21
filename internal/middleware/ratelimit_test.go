package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"relayhub/internal/middleware"
	"relayhub/internal/store"
)

func TestRateLimiter_UnderLimit_Allowed(t *testing.T) {
	t.Parallel()
	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)
	for i := 0; i < 5; i++ {
		limiter.Record("tenant-a")
	}

	allowed, remaining, _ := limiter.Check("tenant-a", 10)
	if !allowed {
		t.Fatal("expected allowed=true when under limit")
	}
	if remaining != 4 {
		t.Errorf("expected remaining=4, got %d", remaining)
	}
}

func TestRateLimiter_AtLimit_Denied(t *testing.T) {
	t.Parallel()
	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)

	for i := 0; i < 10; i++ {
		limiter.Record("tenant-a")
	}

	allowed, remaining, resetAt := limiter.Check("tenant-a", 10)
	if allowed {
		t.Fatal("expected allowed=false when at limit")
	}
	if remaining != 0 {
		t.Errorf("expected remaining=0, got %d", remaining)
	}
	if resetAt.IsZero() {
		t.Error("expected non-zero resetAt")
	}
}

func TestRateLimiter_CrossTenant_Isolation(t *testing.T) {
	t.Parallel()
	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)

	for i := 0; i < 10; i++ {
		limiter.Record("tenant-a")
	}

	allowed, _, _ := limiter.Check("tenant-b", 10)
	if !allowed {
		t.Error("tenant-b must not be affected by tenant-a's usage")
	}
}

func TestRateLimiter_FreshTenant_FullLimit(t *testing.T) {
	t.Parallel()
	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)

	allowed, remaining, _ := limiter.Check("new-tenant", 100)
	if !allowed {
		t.Fatal("fresh tenant should be allowed")
	}
	if remaining != 99 {
		t.Errorf("expected remaining=99, got %d", remaining)
	}
}

func makeTenantCtxRequest(method, path string, t store.TenantRecord) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := middleware.WithTenant(req.Context(), t)
	return req.WithContext(ctx)
}

func TestRateLimitMiddleware_4xxDoesNotConsumeSlot(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)
	rl := middleware.RateLimit(limiter)

	tenant := store.TenantRecord{ID: "t-400", Plan: "free"}

	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	for i := 0; i < 100; i++ {
		req := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenant)
		w := httptest.NewRecorder()
		rl(badHandler).ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("request %d: expected 400, got %d", i, w.Code)
		}
	}

	allowed, _, _ := limiter.Check("t-400", 100)
	if !allowed {
		t.Error("tenant should still be under limit after 100 4xx responses")
	}
}

func TestRateLimitMiddleware_ExceedsLimit_Returns429(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)
	rl := middleware.RateLimit(limiter)

	tenant := store.TenantRecord{ID: "t-429", Plan: "free"}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	for i := 0; i < 100; i++ {
		req := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenant)
		w := httptest.NewRecorder()
		rl(okHandler).ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("request %d should be 201, got %d", i, w.Code)
		}
	}

	req := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenant)
	w := httptest.NewRecorder()
	rl(okHandler).ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "rate limit exceeded") {
		t.Errorf("expected 'rate limit exceeded' in body, got: %s", body)
	}
	if !strings.Contains(body, "resets_at") {
		t.Errorf("expected 'resets_at' in 429 body, got: %s", body)
	}
}

func TestRateLimitMiddleware_RateLimitHeaders_Present(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)
	rl := middleware.RateLimit(limiter)

	tenant := store.TenantRecord{ID: "t-headers", Plan: "free"}
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenant)
	w := httptest.NewRecorder()
	rl(okHandler).ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header missing")
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("X-RateLimit-Remaining header missing")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset header missing")
	}
}

func TestRateLimitMiddleware_TenantIsolation(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewInMemoryRateLimiter(24 * time.Hour)
	rl := middleware.RateLimit(limiter)

	tenantA := store.TenantRecord{ID: "t-a-iso", Plan: "free"}
	tenantB := store.TenantRecord{ID: "t-b-iso", Plan: "free"}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	for i := 0; i < 100; i++ {
		req := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenantA)
		w := httptest.NewRecorder()
		rl(okHandler).ServeHTTP(w, req)
	}

	reqA := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenantA)
	wA := httptest.NewRecorder()
	rl(okHandler).ServeHTTP(wA, reqA)
	if wA.Code != http.StatusTooManyRequests {
		t.Errorf("tenant A: expected 429, got %d", wA.Code)
	}

	reqB := makeTenantCtxRequest(http.MethodPost, "/v1/notify", tenantB)
	wB := httptest.NewRecorder()
	rl(okHandler).ServeHTTP(wB, reqB)
	if wB.Code != http.StatusCreated {
		t.Errorf("tenant B must not be affected by tenant A's limit; got %d", wB.Code)
	}
}

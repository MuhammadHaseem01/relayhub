package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var planLimits = map[string]int{
	"free": 100,
}

func LimitForPlan(plan string) int {
	if l, ok := planLimits[plan]; ok {
		return l
	}
	return 100
}

type InMemoryRateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	window  time.Duration
}

func NewInMemoryRateLimiter(window time.Duration) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		entries: make(map[string][]time.Time),
		window:  window,
	}
}

func (r *InMemoryRateLimiter) Check(tenantID string, limit int) (allowed bool, remaining int, resetAt time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.prune(tenantID, now.Add(-r.window))

	ts := r.entries[tenantID]
	used := len(ts)

	if used >= limit {
		return false, 0, ts[0].Add(r.window)
	}

	rem := limit - used - 1
	if rem < 0 {
		rem = 0
	}
	var reset time.Time
	if len(ts) > 0 {
		reset = ts[0].Add(r.window)
	} else {
		reset = now.Add(r.window)
	}
	return true, rem, reset
}

func (r *InMemoryRateLimiter) Record(tenantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[tenantID] = append(r.entries[tenantID], time.Now())
}

func (r *InMemoryRateLimiter) prune(tenantID string, cutoff time.Time) {
	ts := r.entries[tenantID]
	i := 0
	for i < len(ts) && ts[i].Before(cutoff) {
		i++
	}
	r.entries[tenantID] = ts[i:]
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func RateLimit(limiter *InMemoryRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, ok := TenantFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			limit := LimitForPlan(tenant.Plan)
			allowed, remaining, resetAt := limiter.Check(tenant.ID, limit)
			resetStr := resetAt.UTC().Format(time.RFC3339)

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", resetStr)
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"success":   false,
					"error":     "rate limit exceeded — free plan allows 100 notifications per 24-hour rolling window",
					"limit":     limit,
					"remaining": 0,
					"resets_at": resetStr,
				})
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", resetStr)

			rec := newStatusRecorder(w)
			next.ServeHTTP(rec, r)

			if rec.statusCode < 400 || rec.statusCode >= 500 {
				limiter.Record(tenant.ID)
			}
		})
	}
}

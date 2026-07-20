package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"relayhub/internal/store"

	"github.com/jackc/pgx/v5"
)

type contextKey string

const TenantContextKey contextKey = "tenant"

func Auth(db *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				writeUnauthorized(w, "X-API-Key header is required")
				return
			}

			tenant, err := db.GetTenantByAPIKey(r.Context(), apiKey)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) || isNotFound(err) {
					writeUnauthorized(w, "invalid API key")
					return
				}
				writeUnauthorized(w, "invalid API key")
				return
			}

			ctx := context.WithValue(r.Context(), TenantContextKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TenantFromContext(ctx context.Context) (store.TenantRecord, bool) {
	t, ok := ctx.Value(TenantContextKey).(store.TenantRecord)
	return t, ok
}

func WithTenant(ctx context.Context, t store.TenantRecord) context.Context {
	return context.WithValue(ctx, TenantContextKey, t)
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
	})
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows) ||
		containsString(err.Error(), "not found")
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

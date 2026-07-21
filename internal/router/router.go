package router

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"relayhub/internal/middleware"
	"relayhub/internal/service/notify_service"
	"relayhub/internal/store"

	"github.com/google/uuid"
)

type Config struct {
	NotifyService notify_service.NotifyService
	Store         *store.Store
	Logger        *slog.Logger
}

type Server struct {
	notify      notify_service.NotifyService
	store       *store.Store
	logger      *slog.Logger
	rateLimiter *middleware.InMemoryRateLimiter
}

func New(cfg Config) http.Handler {
	s := &Server{
		notify:      cfg.NotifyService,
		store:       cfg.Store,
		logger:      cfg.Logger,
		rateLimiter: middleware.NewInMemoryRateLimiter(24 * time.Hour),
	}
	return s.withMiddleware(s.routes())
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/tenants", s.handleRegisterTenant)

	auth := middleware.Auth(s.store)
	rl := middleware.RateLimit(s.rateLimiter)

	mux.Handle("POST /v1/notify", auth(rl(http.HandlerFunc(s.handleSend))))

	mux.Handle("GET /v1/logs", auth(http.HandlerFunc(s.handleLogs)))
	mux.Handle("GET /v1/usage", auth(http.HandlerFunc(s.handleUsage)))

	return mux
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > 1 && strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path = strings.TrimRight(r.URL.Path, "/")
		}

		w.Header().Set("X-Request-ID", uuid.New().String())

		s.logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{"status": "ok", "service": "relayhub"})
}

type registerTenantRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleRegisterTenant(w http.ResponseWriter, r *http.Request) {
	var req registerTenantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		s.logger.Error("failed to generate API key", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	tenant, err := s.store.CreateTenant(r.Context(), req.Name, apiKey)
	if err != nil {
		s.logger.Error("failed to create tenant", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}

	s.logger.Info("tenant registered", "tenant_id", tenant.ID, "name", tenant.Name)
	writeCreated(w, map[string]string{
		"tenant_id": tenant.ID,
		"api_key":   tenant.APIKey,
	})
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand read: %w", err)
	}
	return "rh_" + hex.EncodeToString(b), nil
}

type notifyRequest struct {
	Recipient string `json:"recipient"`

	DiscordRecipient string `json:"discord_recipient"`
	EmailRecipient   string `json:"email_recipient"`

	Message        string `json:"message"`
	Channel        string `json:"channel"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req notifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if req.Channel == "" {
		writeError(w, http.StatusBadRequest, "channel is required")
		return
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = r.Header.Get("X-Idempotency-Key")
	}
	switch req.Channel {
	case "auto":
		if req.DiscordRecipient == "" || req.EmailRecipient == "" {
			writeError(w, http.StatusBadRequest, "auto channel requires discord_recipient and email_recipient")
			return
		}
	case "discord", "email":
		if req.Recipient == "" {
			writeError(w, http.StatusBadRequest, "recipient is required for channel="+req.Channel)
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "unsupported channel: "+req.Channel+" — supported: discord, email, auto")
		return
	}

	resp, err := s.notify.Send(r.Context(), notify_service.Request{
		TenantID:         tenant.ID,
		Recipient:        req.Recipient,
		DiscordRecipient: req.DiscordRecipient,
		EmailRecipient:   req.EmailRecipient,
		Message:          req.Message,
		Channel:          req.Channel,
		IdempotencyKey:   req.IdempotencyKey,
	})

	if err != nil && err.Error() == "idempotency: a request with this key is currently being processed" {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"success": false,
			"data":    resp,
		})
		return
	}

	writeCreated(w, resp)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := queryInt(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}

	logs, err := s.store.GetLogs(r.Context(), tenant.ID, limit)
	if err != nil {
		s.logger.Error("failed to fetch logs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch logs")
		return
	}

	writeOK(w, map[string]any{
		"count": len(logs),
		"logs":  logs,
	})
}

// ── GET /v1/usage ─────────────────────────────────────────────────────────────

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	planLimit := middleware.LimitForPlan(tenant.Plan)

	// Query the DB for accurate counts — this reflects real usage even after
	// a process restart when the in-memory limiter counter has been lost.
	usage, err := s.store.GetTenantUsage(r.Context(), tenant.ID)
	if err != nil {
		s.logger.Error("failed to get tenant usage", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get usage stats")
		return
	}

	remaining := planLimit - usage.Count
	if remaining < 0 {
		remaining = 0
	}

	// resets_at: when the oldest notification in the window expires.
	// If no notifications yet, report 24h from now.
	var resetsAt string
	if usage.OldestAt != nil {
		resetsAt = usage.OldestAt.Add(24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
	} else {
		resetsAt = time.Now().Add(24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
	}

	writeOK(w, map[string]any{
		"tenant_id": tenant.ID,
		"plan":      tenant.Plan,
		"limit":     planLimit,
		"used":      usage.Count,
		"remaining": remaining,
		"resets_at": resetsAt,
	})
}

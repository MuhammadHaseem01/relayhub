package router

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"relayhub/internal/health"
	"relayhub/internal/middleware"
	"relayhub/internal/queue"
	"relayhub/internal/store"
	"relayhub/internal/webhook"

	"github.com/google/uuid"
)

const maxScheduleAhead = 30 * 24 * time.Hour

type Config struct {
	Store      *store.Store
	Logger     *slog.Logger
	Dispatcher *webhook.Dispatcher
	Queue      *queue.Queue
	IdemStore  store.IdempotencyStore
	Health     *health.Registry
}

type Server struct {
	store       *store.Store
	logger      *slog.Logger
	rateLimiter *middleware.InMemoryRateLimiter
	dispatcher  *webhook.Dispatcher
	queue       *queue.Queue
	idemStore   store.IdempotencyStore
	health      *health.Registry
}

func New(cfg Config) http.Handler {
	s := &Server{
		store:       cfg.Store,
		logger:      cfg.Logger,
		rateLimiter: middleware.NewInMemoryRateLimiter(24 * time.Hour),
		dispatcher:  cfg.Dispatcher,
		queue:       cfg.Queue,
		idemStore:   cfg.IdemStore,
		health:      cfg.Health,
	}
	return s.withMiddleware(s.routes())
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/health/providers", s.handleProviderHealth)
	mux.HandleFunc("POST /v1/tenants", s.handleRegisterTenant)

	auth := middleware.Auth(s.store)
	rl := middleware.RateLimit(s.rateLimiter)

	mux.Handle("POST /v1/notify", auth(rl(http.HandlerFunc(s.handleSend))))
	mux.Handle("GET /v1/notify/dead-letter", auth(http.HandlerFunc(s.handleListDeadLetter)))
	mux.Handle("POST /v1/notify/{request_id}/replay", auth(http.HandlerFunc(s.handleReplayNotification)))
	mux.Handle("GET /v1/notify/{request_id}", auth(http.HandlerFunc(s.handleGetNotification)))
	mux.Handle("DELETE /v1/notify/{request_id}", auth(http.HandlerFunc(s.handleCancelNotification)))

	mux.Handle("GET /v1/logs", auth(http.HandlerFunc(s.handleLogs)))
	mux.Handle("GET /v1/usage", auth(http.HandlerFunc(s.handleUsage)))

	mux.Handle("POST /v1/templates", auth(http.HandlerFunc(s.handleCreateTemplate)))
	mux.Handle("GET /v1/templates", auth(http.HandlerFunc(s.handleListTemplates)))
	mux.Handle("GET /v1/templates/{name}", auth(http.HandlerFunc(s.handleGetTemplate)))
	mux.Handle("PUT /v1/templates/{name}", auth(http.HandlerFunc(s.handleUpdateTemplate)))
	mux.Handle("DELETE /v1/templates/{name}", auth(http.HandlerFunc(s.handleDeleteTemplate)))

	mux.Handle("PUT /v1/webhook", auth(http.HandlerFunc(s.handleSetWebhook)))
	mux.Handle("DELETE /v1/webhook", auth(http.HandlerFunc(s.handleDeleteWebhook)))
	mux.Handle("GET /v1/webhook/deliveries", auth(http.HandlerFunc(s.handleGetWebhookDeliveries)))

	return mux
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:5173"
		}
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, o := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(o) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Idempotency-Key")
					break
				}
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

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

func (s *Server) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		writeOK(w, map[string]string{
			"discord": "healthy",
			"email":   "healthy",
			"smtp":    "healthy",
		})
		return
	}
	writeOK(w, s.health.Snapshot())
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

var templateNameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

const maxTemplateBodyLen = 4000

type notifyRequest struct {
	Recipient string `json:"recipient"`

	DiscordRecipient string `json:"discord_recipient"`
	EmailRecipient   string `json:"email_recipient"`

	Message        string `json:"message"`
	Channel        string `json:"channel"`
	IdempotencyKey string `json:"idempotency_key"`

	Template  string            `json:"template"`
	Variables map[string]string `json:"variables"`

	SendAt string `json:"send_at"`
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
	if req.Message != "" && req.Template != "" {
		writeError(w, http.StatusBadRequest, "provide either 'message' or 'template', not both")
		return
	}

	if req.Template != "" {
		tmpl, err := s.store.GetTemplate(r.Context(), tenant.ID, req.Template)
		if err != nil {
			if errors.Is(err, store.ErrTemplateNotFound) {
				writeError(w, http.StatusNotFound, fmt.Sprintf("template %q not found", req.Template))
				return
			}
			s.logger.Error("template lookup failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to load template")
			return
		}

		vars := req.Variables
		if vars == nil {
			vars = map[string]string{}
		}

		rendered, missing, err := store.SubstituteVars(tmpl.Body, vars)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success":           false,
				"error":             "template is missing required variables",
				"missing_variables": missing,
			})
			return
		}
		req.Message = rendered
	}

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required (or provide a template)")
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
	case "discord", "email", "smtp":
		if req.Recipient == "" {
			writeError(w, http.StatusBadRequest, "recipient is required for channel="+req.Channel)
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "unsupported channel: "+req.Channel+" — supported: discord, email, smtp, auto")
		return
	}

	if req.SendAt != "" {
		sendAt, err := time.Parse(time.RFC3339, req.SendAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "send_at must be a valid RFC3339 timestamp (e.g. 2026-07-25T09:00:00Z)")
			return
		}
		if sendAt.After(time.Now().Add(maxScheduleAhead)) {
			writeError(w, http.StatusBadRequest, "send_at must not be more than 30 days in the future")
			return
		}
		if sendAt.After(time.Now()) {
			requestID := uuid.New().String()
			rec, err := s.store.CreateScheduled(r.Context(), store.NotificationRecord{
				TenantID:         tenant.ID,
				RequestID:        requestID,
				Recipient:        req.Recipient,
				Channel:          req.Channel,
				Message:          req.Message,
				IdempotencyKey:   req.IdempotencyKey,
				ScheduledFor:     &sendAt,
				DiscordRecipient: req.DiscordRecipient,
				EmailRecipient:   req.EmailRecipient,
			})
			if err != nil {
				s.logger.Error("failed to schedule notification", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to schedule notification")
				return
			}
			s.logger.Info("notification scheduled", "request_id", requestID, "send_at", sendAt)
			writeJSON(w, http.StatusAccepted, map[string]any{
				"success": true,
				"data": map[string]any{
					"request_id":    rec.RequestID,
					"status":        "scheduled",
					"scheduled_for": sendAt.UTC().Format(time.RFC3339),
				},
			})
			return
		}
	}

	requestID := uuid.New().String()
	log := s.logger.With("request_id", requestID, "channel", req.Channel)

	if req.IdempotencyKey != "" {
		record, exists := s.idemStore.GetOrCreate(req.IdempotencyKey)
		if exists {
			if record.InProgress {
				writeError(w, http.StatusConflict,
					"idempotency: a request with this key is currently being processed")
				return
			}
			log.Info("serving from idempotency cache", "key", req.IdempotencyKey)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			_, _ = w.Write(record.Body)
			return
		}
	}

	queuedRec, err := s.store.CreateQueued(r.Context(), store.NotificationRecord{
		TenantID:         tenant.ID,
		RequestID:        requestID,
		Recipient:        req.Recipient,
		Channel:          req.Channel,
		Message:          req.Message,
		IdempotencyKey:   req.IdempotencyKey,
		DiscordRecipient: req.DiscordRecipient,
		EmailRecipient:   req.EmailRecipient,
	})
	if err != nil {
		log.Error("failed to write queued row", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to accept notification")
		return
	}

	if err := s.queue.Enqueue(r.Context(), queue.NotificationJob{
		RequestID:        queuedRec.RequestID,
		TenantID:         tenant.ID,
		Channel:          req.Channel,
		Recipient:        req.Recipient,
		Message:          req.Message,
		IdempotencyKey:   req.IdempotencyKey,
		DiscordRecipient: req.DiscordRecipient,
		EmailRecipient:   req.EmailRecipient,
	}); err != nil {
		log.Error("failed to enqueue notification", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to enqueue notification")
		return
	}

	log.Info("notification queued")

	respBody := map[string]any{
		"success": true,
		"data": map[string]any{
			"request_id": requestID,
			"status":     "queued",
		},
	}

	if req.IdempotencyKey != "" {
		body, _ := json.Marshal(respBody)
		s.idemStore.Save(req.IdempotencyKey, http.StatusCreated, body)
	}

	writeCreated(w, map[string]any{
		"request_id": requestID,
		"status":     "queued",
	})
}

func (s *Server) handleGetNotification(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reqID := r.PathValue("request_id")
	rec, err := s.store.GetNotificationByRequestID(r.Context(), tenant.ID, reqID)
	if err != nil {
		if errors.Is(err, store.ErrNotificationNotFound) {
			writeError(w, http.StatusNotFound, "notification not found: "+reqID)
			return
		}
		s.logger.Error("get notification failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get notification")
		return
	}
	writeOK(w, rec)
}

func (s *Server) handleCancelNotification(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reqID := r.PathValue("request_id")
	if err := s.store.CancelScheduledNotification(r.Context(), tenant.ID, reqID); err != nil {
		switch {
		case errors.Is(err, store.ErrNotificationNotFound):
			writeError(w, http.StatusNotFound, "notification not found: "+reqID)
		case errors.Is(err, store.ErrNotificationAlreadySent):
			writeError(w, http.StatusConflict, "notification has already been sent or cancelled")
		default:
			s.logger.Error("cancel notification failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to cancel notification")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListDeadLetter(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := queryInt(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}

	records, err := s.store.GetDeadLetterNotifications(r.Context(), tenant.ID, limit)
	if err != nil {
		s.logger.Error("failed to get dead letter notifications", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch dead letter notifications")
		return
	}

	writeOK(w, map[string]any{
		"count":         len(records),
		"notifications": records,
	})
}

func (s *Server) handleReplayNotification(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reqID := r.PathValue("request_id")
	rec, err := s.store.ResetDeadLetter(r.Context(), tenant.ID, reqID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotificationNotFound):
			writeError(w, http.StatusNotFound, "notification not found: "+reqID)
		case errors.Is(err, store.ErrNotDeadLetter):
			writeError(w, http.StatusConflict, "notification is not in dead_letter status")
		default:
			s.logger.Error("failed to reset dead letter status", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to replay notification")
		}
		return
	}

	if s.queue != nil {
		if err := s.queue.Enqueue(r.Context(), queue.NotificationJob{
			RequestID:        rec.RequestID,
			TenantID:         rec.TenantID,
			Channel:          rec.Channel,
			Recipient:        rec.Recipient,
			Message:          rec.Message,
			IdempotencyKey:   rec.IdempotencyKey,
			DiscordRecipient: rec.DiscordRecipient,
			EmailRecipient:   rec.EmailRecipient,
			WorkerAttempts:   0,
		}); err != nil {
			s.logger.Error("failed to re-enqueue replayed job", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to enqueue replayed notification")
			return
		}
	}

	s.logger.Info("dead letter notification replayed", "request_id", reqID)
	writeOK(w, map[string]any{
		"request_id": rec.RequestID,
		"status":     "queued",
	})
}

type createTemplateRequest struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if err := validateTemplateName(req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTemplateBody(req.Body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tmpl, err := s.store.CreateTemplate(r.Context(), tenant.ID, req.Name, req.Body)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateTemplate) {
			writeError(w, http.StatusConflict, "a template named "+strQuote(req.Name)+" already exists")
			return
		}
		s.logger.Error("create template failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create template")
		return
	}
	writeCreated(w, tmpl)
}

func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templates, err := s.store.ListTemplates(r.Context(), tenant.ID)
	if err != nil {
		s.logger.Error("list templates failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list templates")
		return
	}
	writeOK(w, map[string]any{
		"count":     len(templates),
		"templates": templates,
	})
}

func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := r.PathValue("name")
	tmpl, err := s.store.GetTemplate(r.Context(), tenant.ID, name)
	if err != nil {
		if errors.Is(err, store.ErrTemplateNotFound) {
			writeError(w, http.StatusNotFound, "template not found: "+name)
			return
		}
		s.logger.Error("get template failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get template")
		return
	}
	writeOK(w, tmpl)
}

type updateTemplateRequest struct {
	Body string `json:"body"`
}

func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateTemplateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if err := validateTemplateBody(req.Body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	name := r.PathValue("name")
	tmpl, err := s.store.UpdateTemplate(r.Context(), tenant.ID, name, req.Body)
	if err != nil {
		if errors.Is(err, store.ErrTemplateNotFound) {
			writeError(w, http.StatusNotFound, "template not found: "+name)
			return
		}
		s.logger.Error("update template failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update template")
		return
	}
	writeOK(w, tmpl)
}

func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := r.PathValue("name")
	if err := s.store.DeleteTemplate(r.Context(), tenant.ID, name); err != nil {
		if errors.Is(err, store.ErrTemplateNotFound) {
			writeError(w, http.StatusNotFound, "template not found: "+name)
			return
		}
		s.logger.Error("delete template failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateTemplateName(name string) error {
	if name == "" {
		return fmt.Errorf("template name is required")
	}
	if !templateNameRe.MatchString(name) {
		return fmt.Errorf("template name must be alphanumeric + underscores only, max 64 characters")
	}
	return nil
}

func validateTemplateBody(body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("template body is required")
	}
	if len(body) > maxTemplateBodyLen {
		return fmt.Errorf("template body exceeds maximum length of %d characters", maxTemplateBodyLen)
	}
	return nil
}

func strQuote(s string) string { return `"` + s + `"` }

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

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	planLimit := middleware.LimitForPlan(tenant.Plan)

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

type setWebhookRequest struct {
	WebhookURL string `json:"webhook_url"`
}

func (s *Server) handleSetWebhook(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.WebhookURL == "" {
		writeError(w, http.StatusBadRequest, "webhook_url is required")
		return
	}
	if !strings.HasPrefix(req.WebhookURL, "https://") {
		writeError(w, http.StatusBadRequest, "webhook_url must use HTTPS")
		return
	}

	secret := tenant.WebhookSecret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			s.logger.Error("failed to generate webhook secret", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to generate webhook secret")
			return
		}
		secret = hex.EncodeToString(b)
	}

	if err := s.store.SetTenantWebhook(r.Context(), tenant.ID, req.WebhookURL, secret); err != nil {
		s.logger.Error("failed to set tenant webhook", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to configure webhook")
		return
	}

	s.logger.Info("webhook configured", "tenant_id", tenant.ID, "url", req.WebhookURL)
	writeOK(w, map[string]string{
		"webhook_url":    req.WebhookURL,
		"webhook_secret": secret,
	})
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := s.store.ClearTenantWebhook(r.Context(), tenant.ID); err != nil {
		s.logger.Error("failed to clear tenant webhook", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to remove webhook")
		return
	}

	s.logger.Info("webhook removed", "tenant_id", tenant.ID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := queryInt(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}

	deliveries, err := s.store.GetWebhookDeliveries(r.Context(), tenant.ID, limit)
	if err != nil {
		s.logger.Error("failed to get webhook deliveries", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch webhook deliveries")
		return
	}

	writeOK(w, map[string]any{
		"count":      len(deliveries),
		"deliveries": deliveries,
	})
}

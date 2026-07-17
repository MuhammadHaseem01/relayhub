// Package router wires HTTP routes to the service layer.
// Handlers are thin: decode request → call service → write response.
// All business logic lives in the service layer.
package router

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"relayhub/internal/service/notify_service"
	"relayhub/internal/store"
)

// Config holds all dependencies injected into the router.
type Config struct {
	NotifyService notify_service.NotifyService
	Store         *store.Store
	Logger        *slog.Logger
}

// Server holds references to all services. Handler methods hang off this struct.
type Server struct {
	notify notify_service.NotifyService
	store  *store.Store
	logger *slog.Logger
}

// New builds the HTTP router and returns its handler.
func New(cfg Config) http.Handler {
	s := &Server{
		notify: cfg.NotifyService,
		store:  cfg.Store,
		logger: cfg.Logger,
	}
	return s.withMiddleware(s.routes())
}

// routes registers all API routes.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/notify", s.handleSend)
	mux.HandleFunc("GET /v1/logs", s.handleLogs)

	return mux
}

// withMiddleware applies top-level middleware (CORS, trailing-slash trim, request ID).
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip trailing slashes
		if len(r.URL.Path) > 1 && strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path = strings.TrimRight(r.URL.Path, "/")
		}

		// Attach a request-scoped request ID to every response
		w.Header().Set("X-Request-ID", uuid.New().String())

		// Structured access log
		s.logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(w, r)
	})
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{"status": "ok", "service": "relayhub"})
}

// notifyRequest mirrors the public API contract for POST /v1/notify.
type notifyRequest struct {
	// For channel="discord" or channel="email" — pass a single recipient.
	Recipient string `json:"recipient"`

	// For channel="auto" — pass both so Discord can fall back to Email.
	DiscordRecipient string `json:"discord_recipient"`
	EmailRecipient   string `json:"email_recipient"`

	Message        string `json:"message"`
	Channel        string `json:"channel"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	var req notifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// Basic validation
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if req.Channel == "" {
		writeError(w, http.StatusBadRequest, "channel is required")
		return
	}

	// Header takes precedence if body field is empty
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = r.Header.Get("X-Idempotency-Key")
	}

	// Per-channel recipient validation
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
		Recipient:        req.Recipient,
		DiscordRecipient: req.DiscordRecipient,
		EmailRecipient:   req.EmailRecipient,
		Message:          req.Message,
		Channel:          req.Channel,
		IdempotencyKey:   req.IdempotencyKey,
	})

	// Idempotency conflict
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
	limit := queryInt(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}

	logs, err := s.store.GetLogs(r.Context(), limit)
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

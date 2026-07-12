package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"relayhub/internal/providers"
	"relayhub/internal/store"
)

// NotifyHandler handles POST /v1/notify and GET /v1/logs.
// It holds a registry of providers keyed by their Name() value,
// so routing to the right provider is a simple map lookup.
type NotifyHandler struct {
	providers map[string]providers.Sender
	store     *store.Store
	logger    *slog.Logger
}

// NewNotifyHandler builds the handler and registers all provided Senders.
func NewNotifyHandler(senders []providers.Sender, s *store.Store, logger *slog.Logger) *NotifyHandler {
	m := make(map[string]providers.Sender, len(senders))
	for _, p := range senders {
		m[p.Name()] = p
	}
	return &NotifyHandler{providers: m, store: s, logger: logger}
}

// ── Request / Response types ──────────────────────────────────────────────────

type notifyRequest struct {
	Recipient string `json:"recipient" binding:"required"`
	Message   string `json:"message"   binding:"required"`
	Channel   string `json:"channel"   binding:"required"`
}

type notifyResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`           // "delivered" | "failed"
	Channel   string `json:"channel"`
	Error     string `json:"error,omitempty"`  // only present on failure
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// Send handles POST /v1/notify
// Accepts a JSON body with recipient, message, and channel.
// Routes to the matching provider, logs the result, and returns a request_id
// for traceability.
func (h *NotifyHandler) Send(c *gin.Context) {
	requestID := uuid.New().String()
	log := h.logger.With("request_id", requestID)

	var req notifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"request_id": requestID,
			"error":      "invalid request body: " + err.Error(),
		})
		return
	}

	log.Info("notify request received",
		"channel", req.Channel,
		"recipient", req.Recipient,
	)

	provider, ok := h.providers[req.Channel]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"request_id": requestID,
			"error":      "unsupported channel: " + req.Channel + " — supported: telegram",
		})
		return
	}

	sendErr := provider.Send(req.Recipient, req.Message)

	status := "delivered"
	errMsg := ""
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
		log.Error("notification failed",
			"channel", req.Channel,
			"error", sendErr,
		)
	} else {
		log.Info("notification delivered", "channel", req.Channel)
	}

	// Persist the log record — failure here is non-fatal for the API response
	logCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if logErr := h.store.LogNotification(logCtx, store.NotificationRecord{
		RequestID:    requestID,
		Recipient:    req.Recipient,
		Channel:      req.Channel,
		Message:      req.Message,
		Status:       status,
		ErrorMessage: errMsg,
	}); logErr != nil {
		log.Warn("failed to write notification log", "error", logErr)
	}

	if sendErr != nil {
		c.JSON(http.StatusBadGateway, notifyResponse{
			RequestID: requestID,
			Status:    status,
			Channel:   req.Channel,
			Error:     errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, notifyResponse{
		RequestID: requestID,
		Status:    status,
		Channel:   req.Channel,
	})
}

// Logs handles GET /v1/logs
// Returns recent notification delivery records, newest first.
// Optional query param: ?limit=N (default 50, max 200)
func (h *NotifyHandler) Logs(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logs, err := h.store.GetLogs(ctx, limit)
	if err != nil {
		h.logger.Error("failed to fetch logs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(logs),
		"logs":  logs,
	})
}

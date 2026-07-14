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
	"relayhub/internal/retry"
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
	Message           string `json:"message"   binding:"required"`
	Channel           string `json:"channel"   binding:"required"`
	Recipient         string `json:"recipient"`
	TelegramRecipient string `json:"telegram_recipient"`
	EmailRecipient    string `json:"email_recipient"`
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

	if req.Channel == "auto" {
		if req.TelegramRecipient == "" || req.EmailRecipient == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"request_id": requestID,
				"error":      "channel 'auto' requires both telegram_recipient and email_recipient",
			})
			return
		}
	} else {
		if req.Recipient == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"request_id": requestID,
				"error":      "recipient is required for channel: " + req.Channel,
			})
			return
		}
	}

	log.Info("notify request received",
		"channel", req.Channel,
		"recipient", req.Recipient,
		"telegram_recipient", req.TelegramRecipient,
		"email_recipient", req.EmailRecipient,
	)

	var status = "delivered"
	var errMsg = ""
	var totalAttempts = 0
	var fallbackUsed = false
	var sendErr error
	var finalRecipient = req.Recipient

	if req.Channel == "auto" {
		telegramProvider, ok1 := h.providers["telegram"]
		emailProvider, ok2 := h.providers["email"]
		if !ok1 || !ok2 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"request_id": requestID,
				"error":      "auto channel requires both telegram and email providers to be configured",
			})
			return
		}

		finalRecipient = req.TelegramRecipient
		sendErr = retry.WithRetry(func() error {
			totalAttempts++
			return telegramProvider.Send(req.TelegramRecipient, req.Message)
		}, 3)

		if sendErr != nil {
			fallbackUsed = true
			log.Warn("telegram failed after retries, falling back to email", "error", sendErr)
			
			finalRecipient = req.EmailRecipient
			emailAttempts := 0
			sendErr = retry.WithRetry(func() error {
				emailAttempts++
				return emailProvider.Send(req.EmailRecipient, req.Message)
			}, 3)
			totalAttempts += emailAttempts
		}

	} else {
		provider, ok := h.providers[req.Channel]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"request_id": requestID,
				"error":      "unsupported channel: " + req.Channel + " — supported: telegram, email, auto",
			})
			return
		}

		sendErr = retry.WithRetry(func() error {
			totalAttempts++
			return provider.Send(req.Recipient, req.Message)
		}, 3)
	}

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
		Recipient:    finalRecipient,
		Channel:      req.Channel,
		Message:      req.Message,
		Status:       status,
		ErrorMessage: errMsg,
		Attempts:     totalAttempts,
		FallbackUsed: fallbackUsed,
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

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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
	providers   map[string]providers.Sender
	store       *store.Store
	idempotency store.IdempotencyStore
	logger      *slog.Logger
}

// NewNotifyHandler builds the handler and registers all provided Senders.
func NewNotifyHandler(senders []providers.Sender, s *store.Store, idem store.IdempotencyStore, logger *slog.Logger) *NotifyHandler {
	m := make(map[string]providers.Sender, len(senders))
	for _, p := range senders {
		m[p.Name()] = p
	}
	return &NotifyHandler{providers: m, store: s, idempotency: idem, logger: logger}
}

// ── Request / Response types ──────────────────────────────────────────────────

type notifyRequest struct {
	Recipient         string `json:"recipient"`
	TelegramRecipient string `json:"telegram_recipient"`
	EmailRecipient    string `json:"email_recipient"`
	Message           string `json:"message" binding:"required"`
	Channel           string `json:"channel" binding:"required"`
	IdempotencyKey    string `json:"idempotency_key"`
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

	// Determine idempotency key
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = c.GetHeader("X-Idempotency-Key")
	}

	var wasCached bool

	if idempotencyKey != "" {
		record, exists := h.idempotency.GetOrCreate(idempotencyKey)
		if exists {
			if record.InProgress {
				c.JSON(http.StatusConflict, gin.H{
					"request_id": requestID,
					"error":      "a request with this idempotency key is currently processing",
				})
				return
			}
			// Cached response
			wasCached = true
			log.Info("serving from idempotency cache", "key", idempotencyKey)
			
			// Log the cached response
			dbCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = h.store.LogNotification(dbCtx, store.NotificationRecord{
				RequestID:         requestID,
				Recipient:         req.Recipient, // best effort for cached logs
				Channel:           req.Channel,
				Message:           req.Message,
				Status:            "delivered (cached)",
				IdempotencyKey:    idempotencyKey,
				WasCachedResponse: true,
			})
			
			c.Data(record.StatusCode, "application/json", record.Body)
			return
		}
	}

	// Validate recipients based on channel
	if req.Channel == "auto" {
		if req.TelegramRecipient == "" || req.EmailRecipient == "" {
			c.JSON(http.StatusBadRequest, gin.H{"request_id": requestID, "error": "auto channel requires both telegram_recipient and email_recipient"})
			return
		}
	} else if req.Channel == "telegram" || req.Channel == "email" {
		if req.Recipient == "" {
			c.JSON(http.StatusBadRequest, gin.H{"request_id": requestID, "error": "recipient is required for " + req.Channel})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"request_id": requestID, "error": "unsupported channel: " + req.Channel + " — supported: telegram, email, auto"})
		return
	}

	log.Info("notify request received",
		"channel", req.Channel,
	)

	var sendErr error
	var status string
	var errMsg string
	var finalChannel string
	var fallbackUsed bool
	totalAttempts := 0

	executeProvider := func(channelName string, recipient string) (int, error) {
		provider, ok := h.providers[channelName]
		if !ok {
			return 0, fmt.Errorf("provider not found: %s", channelName)
		}
		
		attempts, err := retry.WithRetry(func() error {
			return provider.Send(recipient, req.Message)
		}, 3, h.logger)
		
		totalAttempts += attempts
		
		// Log the attempt to DB
		logStatus := "delivered"
		logErr := ""
		if err != nil {
			logStatus = "failed"
			logErr = err.Error()
		}
		
		dbCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.store.LogNotification(dbCtx, store.NotificationRecord{
			RequestID:         requestID,
			Recipient:         recipient,
			Channel:           channelName,
			Message:           req.Message,
			Status:            logStatus,
			ErrorMessage:      logErr,
			Attempts:          attempts,
			FallbackUsed:      fallbackUsed,
			IdempotencyKey:    idempotencyKey,
			WasCachedResponse: wasCached,
		})
		
		return attempts, err
	}

	if req.Channel == "auto" {
		finalChannel = "telegram"
		_, sendErr = executeProvider("telegram", req.TelegramRecipient)
		
		if sendErr != nil {
			fallbackUsed = true
			finalChannel = "email"
			log.Warn("telegram failed completely, falling back to email", "request_id", requestID)
			_, sendErr = executeProvider("email", req.EmailRecipient)
		}
	} else {
		finalChannel = req.Channel
		_, sendErr = executeProvider(req.Channel, req.Recipient)
	}

	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
		log.Error("notification failed", "channel", finalChannel, "error", sendErr)
	} else {
		status = "delivered"
		log.Info("notification delivered", "channel", finalChannel)
	}

	var respBody []byte
	if sendErr != nil {
		c.JSON(http.StatusBadGateway, notifyResponse{
			RequestID: requestID,
			Status:    status,
			Channel:   finalChannel,
			Error:     errMsg,
		})
	} else {
		c.JSON(http.StatusOK, notifyResponse{
			RequestID: requestID,
			Status:    status,
			Channel:   finalChannel,
		})
	}
	
	// Save the response to idempotency store if a key was provided
	if idempotencyKey != "" {
		// Create a mock body since gin doesn't easily expose the written body
		// We know exactly what we just wrote
		var statusCode int
		var resp notifyResponse
		
		if sendErr != nil {
			statusCode = http.StatusBadGateway
			resp = notifyResponse{
				RequestID: requestID,
				Status:    status,
				Channel:   finalChannel,
				Error:     errMsg,
			}
		} else {
			statusCode = http.StatusOK
			resp = notifyResponse{
				RequestID: requestID,
				Status:    status,
				Channel:   finalChannel,
			}
		}
		
		respBody, _ = json.Marshal(resp)
		h.idempotency.Save(idempotencyKey, statusCode, respBody)
	}
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

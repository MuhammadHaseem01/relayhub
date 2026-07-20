package notify_service_impl

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"relayhub/internal/providers"
	"relayhub/internal/service/notify_service"
	"relayhub/internal/store"

	"github.com/google/uuid"
)

type RetryFunc func(fn func() error, maxAttempts int, logger *slog.Logger) (int, error)

type Params struct {
	Providers   []providers.Sender
	Store       *store.Store
	IdemStore   store.IdempotencyStore
	Logger      *slog.Logger
	MaxAttempts int
	Retry       RetryFunc
}

type service struct {
	providers   map[string]providers.Sender
	store       *store.Store
	idemStore   store.IdempotencyStore
	logger      *slog.Logger
	maxAttempts int
	retry       RetryFunc
}

func New(p Params) notify_service.NotifyService {
	m := make(map[string]providers.Sender, len(p.Providers))
	for _, pr := range p.Providers {
		m[pr.Name()] = pr
	}
	return &service{
		providers:   m,
		store:       p.Store,
		idemStore:   p.IdemStore,
		logger:      p.Logger,
		maxAttempts: p.MaxAttempts,
		retry:       p.Retry,
	}
}

func (s *service) Send(ctx context.Context, req notify_service.Request) (notify_service.Response, error) {
	requestID := uuid.New().String()
	log := s.logger.With("request_id", requestID)

	if req.IdempotencyKey != "" {
		record, exists := s.idemStore.GetOrCreate(req.IdempotencyKey)
		if exists {
			if record.InProgress {
				return notify_service.Response{}, fmt.Errorf("idempotency: a request with this key is currently being processed")
			}
			// Return the cached response
			log.Info("serving from idempotency cache", "key", req.IdempotencyKey)
			var cached notify_service.Response
			_ = json.Unmarshal(record.Body, &cached)
			cached.WasCached = true

			s.logToDB(ctx, req.TenantID, requestID, req.Recipient, req.Channel, req.Message,
				"delivered (cached)", "", 0, false, req.IdempotencyKey, true)

			return cached, nil
		}
	}

	var (
		sendErr       error
		finalChannel  string
		fallbackUsed  bool
		totalAttempts int
	)

	execute := func(channelName, recipient string) error {
		pr, ok := s.providers[channelName]
		if !ok {
			return fmt.Errorf("no provider registered for channel %q", channelName)
		}

		attempts, err := s.retry(func() error {
			return pr.Send(recipient, req.Message)
		}, s.maxAttempts, log)

		totalAttempts += attempts

		logStatus := "delivered"
		logErr := ""
		if err != nil {
			logStatus = "failed"
			logErr = err.Error()
		}
		s.logToDB(ctx, req.TenantID, requestID, recipient, channelName, req.Message,
			logStatus, logErr, attempts, fallbackUsed, req.IdempotencyKey, false)

		return err
	}

	switch req.Channel {
	case "auto":
		finalChannel = "discord"
		sendErr = execute("discord", req.DiscordRecipient)
		if sendErr != nil {
			fallbackUsed = true
			finalChannel = "email"
			log.Warn("discord failed, falling back to email", "error", sendErr)
			sendErr = execute("email", req.EmailRecipient)
		}
	default:
		finalChannel = req.Channel
		sendErr = execute(req.Channel, req.Recipient)
	}

	resp := notify_service.Response{
		RequestID: requestID,
		Channel:   finalChannel,
		Status:    "delivered",
	}
	if sendErr != nil {
		resp.Status = "failed"
		resp.Error = sendErr.Error()
		log.Error("notification failed", "channel", finalChannel, "error", sendErr)
	} else {
		log.Info("notification delivered", "channel", finalChannel)
	}

	if req.IdempotencyKey != "" {
		body, _ := json.Marshal(resp)
		statusCode := http.StatusOK
		if sendErr != nil {
			statusCode = http.StatusBadGateway
		}
		s.idemStore.Save(req.IdempotencyKey, statusCode, body)
	}

	return resp, sendErr
}

func (s *service) logToDB(
	ctx context.Context,
	tenantID, requestID, recipient, channel, message, status, errMsg string,
	attempts int, fallbackUsed bool, idempotencyKey string, wasCached bool,
) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := s.store.LogNotification(dbCtx, store.NotificationRecord{
		TenantID:          tenantID,
		RequestID:         requestID,
		Recipient:         recipient,
		Channel:           channel,
		Message:           message,
		Status:            status,
		ErrorMessage:      errMsg,
		Attempts:          attempts,
		FallbackUsed:      fallbackUsed,
		IdempotencyKey:    idempotencyKey,
		WasCachedResponse: wasCached,
	}); err != nil {
		s.logger.Warn("failed to write notification log", "error", err)
	}
}

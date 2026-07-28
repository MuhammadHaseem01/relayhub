package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"relayhub/internal/providers"
	"relayhub/internal/store"
	"relayhub/internal/webhook"
)

type RetryFunc func(fn func() error, maxAttempts int, logger *slog.Logger) (int, error)

type Job struct {
	RequestID        string
	TenantID         string
	Channel          string
	Recipient        string
	Message          string
	DiscordRecipient string
	EmailRecipient   string
}

type Executor struct {
	Providers   map[string]providers.Sender
	Retry       RetryFunc
	MaxAttempts int
	Store       *store.Store
	Dispatcher  *webhook.Dispatcher
	Logger      *slog.Logger
}

func (e *Executor) Run(ctx context.Context, job Job) error {
	log := e.Logger.With("request_id", job.RequestID, "channel", job.Channel)

	var (
		sendErr       error
		finalChannel  = job.Channel
		fallbackUsed  bool
		totalAttempts int
	)

	send := func(channelName, recipient string) error {
		pr, ok := e.Providers[channelName]
		if !ok {
			return fmt.Errorf("dispatch: no provider registered for channel %q", channelName)
		}
		attempts, err := e.Retry(func() error {
			return pr.Send(recipient, job.Message)
		}, e.MaxAttempts, log)
		totalAttempts += attempts
		return err
	}

	switch job.Channel {
	case "auto":
		finalChannel = "discord"
		sendErr = send("discord", job.DiscordRecipient)
		if sendErr != nil {
			fallbackUsed = true
			finalChannel = "email"
			log.Warn("discord failed, falling back to email", "error", sendErr)
			sendErr = send("email", job.EmailRecipient)
		}
	default:
		sendErr = send(job.Channel, job.Recipient)
	}

	finalStatus := "delivered"
	finalErrMsg := ""
	if sendErr != nil {
		finalStatus = "failed"
		finalErrMsg = sendErr.Error()
		log.Error("notification failed", "channel", finalChannel, "error", sendErr)
	} else {
		log.Info("notification delivered", "channel", finalChannel)
	}

	if e.Store != nil {
		if err := e.Store.UpdateNotificationStatus(
			ctx, job.RequestID, finalStatus, finalErrMsg, totalAttempts, fallbackUsed,
		); err != nil {
			log.Error("dispatch: failed to update notification status", "error", err)
		}
	}

	if e.Dispatcher != nil && e.Store != nil {
		tenant, err := e.Store.GetTenantByID(ctx, job.TenantID)
		if err != nil {
			log.Warn("dispatch: failed to load tenant for webhook", "error", err)
		} else if tenant.WebhookURL != "" {
			event := "notification.delivered"
			if sendErr != nil {
				event = "notification.failed"
			}
			e.Dispatcher.FireAsync(
				tenant.ID,
				tenant.WebhookURL,
				tenant.WebhookSecret,
				webhook.EventPayload{
					Event:        event,
					RequestID:    job.RequestID,
					ChannelUsed:  finalChannel,
					FallbackUsed: fallbackUsed,
					Attempts:     totalAttempts,
					Timestamp:    time.Now().UTC(),
				},
			)
		}
	}

	return sendErr
}

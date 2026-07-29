package dispatch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"relayhub/internal/health"
	"relayhub/internal/providers"
	"relayhub/internal/queue"
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
	WorkerAttempts   int
}

type Executor struct {
	Providers         map[string]providers.Sender
	Retry             RetryFunc
	MaxAttempts       int
	MaxWorkerAttempts int
	Store             *store.Store
	Dispatcher        *webhook.Dispatcher
	Queue             *queue.Queue
	Health            *health.Registry
	Logger            *slog.Logger
}

func (e *Executor) Run(ctx context.Context, job Job) error {
	log := e.Logger.With("request_id", job.RequestID, "channel", job.Channel)

	maxWorkerAttempts := e.MaxWorkerAttempts
	if maxWorkerAttempts <= 0 {
		maxWorkerAttempts = 3
	}

	if job.WorkerAttempts >= maxWorkerAttempts {
		log.Error("job exceeded max worker attempts, moving to DLQ", "worker_attempts", job.WorkerAttempts)

		if e.Store != nil {
			if err := e.Store.UpdateNotificationStatus(
				ctx, job.RequestID, "dead_letter", "exceeded maximum worker attempts", 0, false,
			); err != nil {
				log.Error("dispatch: failed to update status to dead_letter", "error", err)
			}
		}

		if e.Queue != nil {
			if err := e.Queue.EnqueueDLQ(ctx, queue.NotificationJob{
				RequestID:        job.RequestID,
				TenantID:         job.TenantID,
				Channel:          job.Channel,
				Recipient:        job.Recipient,
				Message:          job.Message,
				DiscordRecipient: job.DiscordRecipient,
				EmailRecipient:   job.EmailRecipient,
				WorkerAttempts:   job.WorkerAttempts,
			}); err != nil {
				log.Error("dispatch: failed to enqueue job to DLQ", "error", err)
			}
		}

		if e.Dispatcher != nil && e.Store != nil {
			if tenant, err := e.Store.GetTenantByID(ctx, job.TenantID); err == nil && tenant.WebhookURL != "" {
				e.Dispatcher.FireAsync(
					tenant.ID,
					tenant.WebhookURL,
					tenant.WebhookSecret,
					webhook.EventPayload{
						Event:        "notification.failed",
						RequestID:    job.RequestID,
						ChannelUsed:  job.Channel,
						FallbackUsed: false,
						Attempts:     0,
						Timestamp:    time.Now().UTC(),
					},
				)
			}
		}

		return fmt.Errorf("job %s dead lettered after %d worker attempts", job.RequestID, job.WorkerAttempts)
	}

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

		if e.Health != nil {
			if err == nil {
				e.Health.RecordSuccess(channelName)
			} else {
				e.Health.RecordFailure(channelName)
			}
		}

		return err
	}

	switch job.Channel {
	case "auto":
		if e.Health != nil && !e.Health.IsHealthy("discord") {
			fallbackUsed = true
			finalChannel = "email"
			log.Warn("discord provider circuit open (unhealthy), going straight to email fallback")
			sendErr = send("email", job.EmailRecipient)
		} else {
			finalChannel = "discord"
			sendErr = send("discord", job.DiscordRecipient)
			if sendErr != nil {
				fallbackUsed = true
				finalChannel = "email"
				log.Warn("discord failed, falling back to email", "error", sendErr)
				sendErr = send("email", job.EmailRecipient)
			}
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

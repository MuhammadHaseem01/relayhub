package scheduler

import (
	"context"
	"log/slog"
	"time"

	"relayhub/internal/queue"
	"relayhub/internal/store"
)

type Scheduler struct {
	store    *store.Store
	queue    *queue.Queue
	interval time.Duration
	logger   *slog.Logger
}

type Params struct {
	Store    *store.Store
	Queue    *queue.Queue
	Interval time.Duration
	Logger   *slog.Logger
}

func New(p Params) *Scheduler {
	interval := p.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &Scheduler{
		store:    p.Store,
		queue:    p.Queue,
		interval: interval,
		logger:   p.Logger,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Info("scheduler started", "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

const claimBatchSize = 10

func (s *Scheduler) tick(ctx context.Context) {
	claimed, err := s.store.ClaimDueNotifications(ctx, claimBatchSize)
	if err != nil {
		s.logger.Error("scheduler: claim failed", "error", err)
		return
	}
	if len(claimed) == 0 {
		return
	}
	s.logger.Info("scheduler: enqueueing due notifications", "count", len(claimed))
	for _, rec := range claimed {
		s.enqueue(ctx, rec)
	}
}

func (s *Scheduler) enqueue(ctx context.Context, rec store.NotificationRecord) {
	log := s.logger.With("request_id", rec.RequestID, "channel", rec.Channel)
	err := s.queue.Enqueue(ctx, queue.NotificationJob{
		RequestID:        rec.RequestID,
		TenantID:         rec.TenantID,
		Channel:          rec.Channel,
		Recipient:        rec.Recipient,
		Message:          rec.Message,
		IdempotencyKey:   rec.IdempotencyKey,
		DiscordRecipient: rec.DiscordRecipient,
		EmailRecipient:   rec.EmailRecipient,
	})
	if err != nil {
		log.Error("scheduler: failed to enqueue due notification", "error", err)
		if uerr := s.store.UpdateNotificationStatus(
			ctx, rec.RequestID, "failed",
			"scheduler: enqueue failed: "+err.Error(), 0, false,
		); uerr != nil {
			log.Error("scheduler: also failed to update status", "error", uerr)
		}
		return
	}
	log.Info("scheduler: notification enqueued for worker pickup")
}

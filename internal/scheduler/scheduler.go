package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"relayhub/internal/providers"
	"relayhub/internal/store"
)

type RetryFunc func(fn func() error, maxAttempts int, logger *slog.Logger) (int, error)

type Scheduler struct {
	store       *store.Store
	providers   map[string]providers.Sender
	retry       RetryFunc
	maxAttempts int
	interval    time.Duration
	logger      *slog.Logger
}

type Params struct {
	Store       *store.Store
	Providers   []providers.Sender
	Retry       RetryFunc
	MaxAttempts int
	Interval    time.Duration
	Logger      *slog.Logger
}

func New(p Params) *Scheduler {
	interval := p.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}
	pm := make(map[string]providers.Sender, len(p.Providers))
	for _, pr := range p.Providers {
		pm[pr.Name()] = pr
	}
	return &Scheduler{
		store:       p.Store,
		providers:   pm,
		retry:       p.Retry,
		maxAttempts: p.MaxAttempts,
		interval:    interval,
		logger:      p.Logger,
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
	s.logger.Info("scheduler: dispatching due notifications", "count", len(claimed))
	for _, rec := range claimed {
		s.dispatch(ctx, rec)
	}
}

func (s *Scheduler) dispatch(ctx context.Context, rec store.NotificationRecord) {
	log := s.logger.With("request_id", rec.RequestID, "channel", rec.Channel)

	var (
		finalStatus   = "delivered"
		finalErr      = ""
		fallbackUsed  = false
		totalAttempts int
	)

	execute := func(channelName, recipient string) error {
		pr, ok := s.providers[channelName]
		if !ok {
			return fmt.Errorf("no provider registered for channel %q", channelName)
		}
		attempts, err := s.retry(func() error {
			return pr.Send(recipient, rec.Message)
		}, s.maxAttempts, log)
		totalAttempts += attempts
		return err
	}

	var sendErr error
	switch rec.Channel {
	case "auto":
		sendErr = execute("discord", rec.DiscordRecipient)
		if sendErr != nil {
			fallbackUsed = true
			log.Warn("discord failed, falling back to email", "error", sendErr)
			sendErr = execute("email", rec.EmailRecipient)
		}
	default:
		sendErr = execute(rec.Channel, rec.Recipient)
	}

	if sendErr != nil {
		finalStatus = "failed"
		finalErr = sendErr.Error()
		log.Error("scheduled notification failed", "error", sendErr)
	} else {
		log.Info("scheduled notification delivered")
	}

	if err := s.store.UpdateNotificationStatus(
		ctx, rec.RequestID, finalStatus, finalErr, totalAttempts, fallbackUsed,
	); err != nil {
		log.Error("scheduler: failed to update notification status", "error", err)
	}
}

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"relayhub/internal/config"
	"relayhub/internal/dispatch"
	"relayhub/internal/providers"
	"relayhub/internal/queue"
	"relayhub/internal/retry"
	"relayhub/internal/router"
	"relayhub/internal/scheduler"
	"relayhub/internal/store"
	"relayhub/internal/webhook"
	"relayhub/internal/worker"
)

func Start(cfg *config.Config, logger *slog.Logger) error {
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	defer db.Close()
	logger.Info("connected to postgres")

	q, err := queue.New(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("queue init: %w", err)
	}
	defer q.Close()
	if err := q.Ping(context.Background()); err != nil {
		return fmt.Errorf("redis unreachable: %w", err)
	}
	if err := q.EnsureGroup(context.Background()); err != nil {
		return fmt.Errorf("redis consumer group init: %w", err)
	}
	logger.Info("connected to redis", "url", cfg.RedisURL)

	discord := providers.NewDiscordProvider(cfg.DiscordWebhookURL)
	email := providers.NewEmailProvider(cfg.ResendAPIKey, cfg.FromEmail)
	smtpProvider := providers.NewSMTPProvider(
		cfg.SMTPHost, cfg.SMTPPort,
		cfg.SMTPUsername, cfg.SMTPPassword,
		cfg.SMTPFrom,
	)
	allProviders := []providers.Sender{discord, email, smtpProvider}
	providerMap := make(map[string]providers.Sender, len(allProviders))
	for _, p := range allProviders {
		providerMap[p.Name()] = p
	}

	whDispatcher := webhook.New(webhook.Params{
		Store:       db,
		Retry:       retry.WithRetry,
		MaxAttempts: 3,
		Logger:      logger,
	})

	executor := &dispatch.Executor{
		Providers:   providerMap,
		Retry:       retry.WithRetry,
		MaxAttempts: 3,
		Store:       db,
		Dispatcher:  whDispatcher,
		Logger:      logger,
	}

	idemStore := store.NewInMemoryIdempotencyStore()

	r := router.New(router.Config{
		Store:      db,
		Logger:     logger,
		Dispatcher: whDispatcher,
		Queue:      q,
		IdemStore:  idemStore,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	sched := scheduler.New(scheduler.Params{
		Store:    db,
		Queue:    q,
		Interval: 30 * time.Second,
		Logger:   logger,
	})
	go sched.Run(bgCtx)

	pool := worker.New(worker.Params{
		WorkerCount: cfg.WorkerCount,
		Queue:       q,
		Executor:    executor,
		Logger:      logger,
	})
	go pool.Run(bgCtx)

	errs := make(chan error, 1)
	go func() {
		logger.Info("RelayHub listening", "addr", srv.Addr)
		errs <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGABRT, os.Interrupt)

	select {
	case err := <-errs:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig)
		bgCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}

	return nil
}

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
	"relayhub/internal/providers"
	"relayhub/internal/retry"
	"relayhub/internal/router"
	"relayhub/internal/scheduler"
	"relayhub/internal/service/notify_service/notify_service_impl"
	"relayhub/internal/store"
)

func Start(cfg *config.Config, logger *slog.Logger) error {
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	defer db.Close()
	logger.Info("connected to postgres")

	idemStore := store.NewInMemoryIdempotencyStore()

	discord := providers.NewDiscordProvider(cfg.DiscordWebhookURL)
	email := providers.NewEmailProvider(cfg.ResendAPIKey, cfg.FromEmail)
	allProviders := []providers.Sender{discord, email}

	notifySvc := notify_service_impl.New(notify_service_impl.Params{
		Providers:   allProviders,
		Store:       db,
		IdemStore:   idemStore,
		Logger:      logger,
		MaxAttempts: 3,
		Retry:       retry.WithRetry,
	})

	r := router.New(router.Config{
		NotifyService: notifySvc,
		Store:         db,
		Logger:        logger,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()

	sched := scheduler.New(scheduler.Params{
		Store:       db,
		Providers:   allProviders,
		Retry:       retry.WithRetry,
		MaxAttempts: 3,
		Interval:    30 * time.Second,
		Logger:      logger,
	})
	go sched.Run(schedCtx)

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
		schedCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}

	return nil
}

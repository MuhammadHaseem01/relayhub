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
	"relayhub/internal/service/notify_service/notify_service_impl"
	"relayhub/internal/store"
)

// Start wires all dependencies, starts the HTTP server, and blocks until a
// shutdown signal is received or a fatal error occurs.
func Start(cfg *config.Config, logger *slog.Logger) error {
	// ── Store ─────────────────────────────────────────────────────────────────
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database init: %w", err)
	}
	defer db.Close()
	logger.Info("connected to postgres")

	idemStore := store.NewInMemoryIdempotencyStore()

	// ── Providers ─────────────────────────────────────────────────────────────
	discord := providers.NewDiscordProvider(cfg.DiscordWebhookURL)
	email := providers.NewEmailProvider(cfg.ResendAPIKey, cfg.FromEmail)

	// ── Services ──────────────────────────────────────────────────────────────
	notifySvc := notify_service_impl.New(notify_service_impl.Params{
		Providers:   []providers.Sender{discord, email},
		Store:       db,
		IdemStore:   idemStore,
		Logger:      logger,
		MaxAttempts: 3,
		Retry:       retry.WithRetry,
	})

	// ── Router ────────────────────────────────────────────────────────────────
	r := router.New(router.Config{
		NotifyService: notifySvc,
		Store:         db,
		Logger:        logger,
	})

	// ── HTTP Server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("RelayHub listening", "addr", srv.Addr)
		errs <- srv.ListenAndServe()
	}()

	// Block until OS signal or server error
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGABRT, os.Interrupt)

	select {
	case err := <-errs:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}

	return nil
}

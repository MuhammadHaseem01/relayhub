package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cargonex-backend/src/config"
	"cargonex-backend/src/database"
	"cargonex-backend/src/router"
)

func Start(cfg config.Config) error {
	db, err := database.New(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	r := router.NewRouter(db, router.ServerConfig{
		AuthSecret: cfg.AuthTokenSecret,
		CORSOrigin: cfg.CORSOrigin,
	})
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           r.Engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		errs <- server.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGABRT, os.Interrupt)

	select {
	case err := <-errs:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}

	return nil
}

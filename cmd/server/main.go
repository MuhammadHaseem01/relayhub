package main

import (
	"log/slog"
	"os"

	"relayhub/internal/config"
	"relayhub/internal/handlers"
	"relayhub/internal/providers"
	"relayhub/internal/store"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to postgres")

	// Initialize providers
	telegram := providers.NewTelegramProvider(cfg.TelegramBotToken)
	email := providers.NewEmailProvider(cfg.ResendAPIKey, cfg.FromEmail)

	// Initialize HTTP handlers
	idemStore := store.NewInMemoryIdempotencyStore()
	notifyHandler := handlers.NewNotifyHandler([]providers.Sender{telegram, email}, db, idemStore, logger)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "relayhub"})
	})

	v1 := router.Group("/v1")
	{
		v1.POST("/notify", notifyHandler.Send)
		v1.GET("/logs", notifyHandler.Logs)
	}

	logger.Info("RelayHub started", "port", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		logger.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
		)
	}
}

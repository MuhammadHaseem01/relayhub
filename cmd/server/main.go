package main

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"relayhub/internal/config"
	"relayhub/internal/handlers"
	"relayhub/internal/providers"
	"relayhub/internal/store"
)

func main() {
	// Structured JSON logger — every log line includes a timestamp and level
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to postgres")

	// ── Providers ─────────────────────────────────────────────────────────────
	// To add a new provider: implement providers.Sender and append it here.
	// No changes needed anywhere else.
	senderList := []providers.Sender{
		providers.NewTelegramProvider(cfg.TelegramBotToken),
	}

	// ── HTTP Router ───────────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))

	// Health check — useful for Docker health checks and load balancers
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "relayhub"})
	})

	notifyHandler := handlers.NewNotifyHandler(senderList, db, logger)

	v1 := router.Group("/v1")
	{
		v1.POST("/notify", notifyHandler.Send)
		v1.GET("/logs", notifyHandler.Logs)
	}

	// ── Start ─────────────────────────────────────────────────────────────────
	logger.Info("RelayHub started", "port", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// requestLogger is a minimal Gin middleware that emits one structured log line
// per HTTP request after the handler chain completes.
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

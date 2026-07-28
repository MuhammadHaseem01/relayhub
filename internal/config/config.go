package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	DiscordWebhookURL string

	// Resend (channel=email)
	ResendAPIKey string
	FromEmail    string

	// SMTP (channel=smtp)
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Redis / async worker (Phase 4)
	RedisURL    string
	WorkerCount int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	wc, _ := strconv.Atoi(getEnv("WORKER_COUNT", "5"))
	if wc <= 0 {
		wc = 5
	}

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		FromEmail:         os.Getenv("FROM_EMAIL"),
		SMTPHost:          os.Getenv("SMTP_HOST"),
		SMTPPort:          getEnv("SMTP_PORT", "587"),
		SMTPUsername:      os.Getenv("SMTP_USERNAME"),
		SMTPPassword:      os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:          os.Getenv("SMTP_FROM"),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379"),
		WorkerCount:       wc,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

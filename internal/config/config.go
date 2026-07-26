package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	DiscordWebhookURL string

	ResendAPIKey string
	FromEmail    string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

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

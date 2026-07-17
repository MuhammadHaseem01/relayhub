package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port              string
	DatabaseURL       string
	DiscordWebhookURL string
	ResendAPIKey      string
	FromEmail         string
}

// Load reads configuration from a .env file (if present) and environment variables.
// Environment variables always take precedence over .env values.
func Load() (*Config, error) {
	// Silently ignore missing .env — it may not exist in production/Docker
	_ = godotenv.Load()

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		FromEmail:         os.Getenv("FROM_EMAIL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	// DiscordWebhookURL is optional — requests can pass the webhook URL directly
	// as the recipient field in the API request.
	if cfg.ResendAPIKey == ""{
		return nil, fmt.Errorf("config: RESEND_API_KEY is required")
	}
	if cfg.FromEmail == "" {
		return nil, fmt.Errorf("config: FROM_EMAIL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

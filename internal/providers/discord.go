package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DiscordProvider delivers notifications via Discord Webhooks.
// Docs: https://discord.com/developers/docs/resources/webhook#execute-webhook
//
// recipient = the full Discord Webhook URL
// (e.g. https://discord.com/api/webhooks/{id}/{token})
// No bot token or server permissions required — just a webhook URL.
type DiscordProvider struct {
	defaultWebhookURL string
	client            *http.Client
}

// NewDiscordProvider creates a DiscordProvider.
// defaultWebhookURL is used when no recipient is explicitly provided.
func NewDiscordProvider(defaultWebhookURL string) *DiscordProvider {
	return &DiscordProvider{
		defaultWebhookURL: defaultWebhookURL,
		client:            &http.Client{Timeout: 10 * time.Second},
	}
}

// Name satisfies the Sender interface. Maps to the "channel" field in requests.
func (d *DiscordProvider) Name() string {
	return "discord"
}

type discordWebhookPayload struct {
	Content string `json:"content"`
}

// Send posts message to the Discord webhook identified by recipient.
// recipient must be a full Discord webhook URL.
// If recipient is empty, the provider's default webhook URL is used.
func (d *DiscordProvider) Send(recipient string, message string) error {
	webhookURL := recipient
	if webhookURL == "" {
		webhookURL = d.defaultWebhookURL
	}
	if webhookURL == "" {
		return fmt.Errorf("discord: no webhook URL provided — pass it as recipient or set DISCORD_WEBHOOK_URL")
	}

	payload, err := json.Marshal(discordWebhookPayload{Content: message})
	if err != nil {
		return fmt.Errorf("discord: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("discord: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord: network error: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success
	if resp.StatusCode == http.StatusNoContent || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("discord: invalid webhook URL — 401 Unauthorized")
	case http.StatusNotFound:
		return fmt.Errorf("discord: webhook not found (404) — check the webhook URL")
	case http.StatusTooManyRequests:
		return fmt.Errorf("discord: rate limited (429) — try again later")
	default:
		return fmt.Errorf("discord: API error (status %d): %s", resp.StatusCode, string(body))
	}
}

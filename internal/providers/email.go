package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmailProvider delivers notifications via the Resend API.
type EmailProvider struct {
	APIKey    string
	FromEmail string
}

func NewEmailProvider(apiKey, fromEmail string) *EmailProvider {
	return &EmailProvider{
		APIKey:    apiKey,
		FromEmail: fromEmail,
	}
}

func (p *EmailProvider) Name() string {
	return "email"
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

func (p *EmailProvider) Send(recipient string, message string) error {
	payload := resendPayload{
		From:    p.FromEmail,
		To:      []string{recipient},
		Subject: "Notification from RelayHub",
		Text:    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: failed to encode payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("email: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("email: network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email: provider rejected request (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

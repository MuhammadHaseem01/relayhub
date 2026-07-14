package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type EmailProvider struct {
	apiKey    string
	fromEmail string
}

func NewEmailProvider(apiKey, fromEmail string) *EmailProvider {
	return &EmailProvider{
		apiKey:    apiKey,
		fromEmail: fromEmail,
	}
}

func (p *EmailProvider) Name() string {
	return "email"
}

type resendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Html    string `json:"html"`
}

type resendError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (p *EmailProvider) Send(recipient string, message string) error {
	reqBody := resendRequest{
		From:    p.fromEmail,
		To:      recipient, // Resend expects a single recipient or array. We pass string for simplicity if Resend supports it, or wrap in array below. Actually, Resend expects string or array of strings.
		Subject: "Notification from RelayHub",
		Html:    fmt.Sprintf("<p>%s</p>", message),
	}

	// Resend strictly expects `to` as an array or a single string.
	// But it's usually better to pass array in JSON if possible. String is also valid in their API.

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("email: failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("email: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("email: network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // Success
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var resErr resendError
	if err := json.Unmarshal(bodyBytes, &resErr); err == nil && resErr.Message != "" {
		return fmt.Errorf("email: resend api error (status %d): %s - %s", resp.StatusCode, resErr.Name, resErr.Message)
	}

	return fmt.Errorf("email: resend api error (status %d): %s", resp.StatusCode, string(bodyBytes))
}

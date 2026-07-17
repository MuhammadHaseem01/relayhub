// Package notify_service defines the contract for sending notifications.
// Core business logic (retry, fallback, idempotency) is hidden behind this
// interface so the router never needs to import providers or retry logic.
package notify_service

import "context"

// Request holds everything needed to send a single notification.
type Request struct {
	// For channel="discord" or channel="email":
	Recipient string

	// For channel="auto" (try Discord first, fall back to Email):
	DiscordRecipient string
	EmailRecipient   string

	Message        string
	Channel        string // "discord" | "email" | "auto"
	IdempotencyKey string // optional; prevents duplicate sends
}

// Response is the structured result returned to the caller.
type Response struct {
	RequestID    string `json:"request_id"`
	Status       string `json:"status"`  // "delivered" | "failed" | "delivered (cached)"
	Channel      string `json:"channel"` // final channel used
	Error        string `json:"error,omitempty"`
	WasCached    bool   `json:"was_cached,omitempty"`
}

// NotifyService is the single entry point for all notification delivery.
// Implementations handle retry, fallback, DB logging, and idempotency
// so that callers (router handlers) only work with Request/Response.
type NotifyService interface {
	Send(ctx context.Context, req Request) (Response, error)
}

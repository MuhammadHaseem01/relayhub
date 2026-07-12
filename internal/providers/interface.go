// Package providers defines the universal contract that every notification
// provider must satisfy. Core business logic depends only on this interface —
// adding a new provider (Email, SMS, Discord, etc.) never requires touching
// any handler or service code.
package providers

// Sender is the single interface every notification provider must implement.
//
// recipient semantics are provider-specific:
//   - Telegram: numeric chat_id  (e.g. "123456789")
//   - Email:    email address    (e.g. "user@example.com")
//   - Discord:  webhook URL      (e.g. "https://discord.com/api/webhooks/...")
type Sender interface {
	// Send delivers message to recipient. Returns a descriptive error on failure.
	Send(recipient string, message string) error

	// Name returns the unique, lowercase identifier for this provider.
	// This value is what callers pass as the "channel" field in /notify requests.
	Name() string
}

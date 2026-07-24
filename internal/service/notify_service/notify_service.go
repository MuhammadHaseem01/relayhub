package notify_service

import "context"

type Request struct {
	TenantID         string
	Recipient        string
	DiscordRecipient string
	EmailRecipient   string

	Message        string
	Channel        string
	IdempotencyKey string

	TenantWebhookURL    string
	TenantWebhookSecret string
}

type Response struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Channel   string `json:"channel"`
	Error     string `json:"error,omitempty"`
	WasCached bool   `json:"was_cached,omitempty"`
}

type NotifyService interface {
	Send(ctx context.Context, req Request) (Response, error)
}

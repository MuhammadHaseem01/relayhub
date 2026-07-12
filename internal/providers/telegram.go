package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TelegramProvider delivers notifications via the Telegram Bot API.
// Docs: https://core.telegram.org/bots/api#sendmessage
type TelegramProvider struct {
	botToken string
	apiBase  string
	client   *http.Client
}

// NewTelegramProvider creates a TelegramProvider using the given bot token.
// Obtain a free token by messaging @BotFather on Telegram.
func NewTelegramProvider(botToken string) *TelegramProvider {
	return &TelegramProvider{
		botToken: botToken,
		apiBase:  "https://api.telegram.org",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name satisfies the Sender interface. Maps to the "channel" field in requests.
func (t *TelegramProvider) Name() string {
	return "telegram"
}

// telegramAPIResponse mirrors the top-level Telegram Bot API response envelope.
type telegramAPIResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"` // populated only on error
	ErrorCode   int    `json:"error_code"`  // populated only on error
}

// Send delivers message to the Telegram chat identified by recipient (chat_id).
// chat_id can be a numeric user ID or a public group username like "@mychannel".
func (t *TelegramProvider) Send(recipient string, message string) error {
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", t.apiBase, t.botToken)

	params := url.Values{}
	params.Set("chat_id", recipient)
	params.Set("text", message)
	params.Set("parse_mode", "HTML") // allows <b>, <i>, <code> in messages

	resp, err := t.client.Post(
		endpoint,
		"application/x-www-form-urlencoded",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return fmt.Errorf("telegram: network error reaching API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("telegram: failed to read API response: %w", err)
	}

	var tgResp telegramAPIResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return fmt.Errorf("telegram: unexpected API response format: %w", err)
	}

	if !tgResp.OK {
		// Provide actionable error messages for common failure codes
		switch tgResp.ErrorCode {
		case 401:
			return fmt.Errorf("telegram: invalid bot token (401 Unauthorized) — check TELEGRAM_BOT_TOKEN")
		case 400:
			return fmt.Errorf("telegram: bad request (400) — invalid chat_id %q: %s", recipient, tgResp.Description)
		case 403:
			return fmt.Errorf("telegram: bot was blocked by the user or is not a member of the chat (403): %s", tgResp.Description)
		default:
			return fmt.Errorf("telegram: API error %d: %s", tgResp.ErrorCode, tgResp.Description)
		}
	}

	return nil
}

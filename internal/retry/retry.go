package retry

import (
	"log/slog"
	"time"
)

// WithRetry wraps a function with exponential backoff retries.
// It uses delays of 1s, 2s, 4s, etc., between attempts.
func WithRetry(fn func() error, maxAttempts int) error {
	var err error
	delay := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil // Success
		}

		if attempt < maxAttempts {
			slog.Warn("action failed, retrying",
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"next_wait", delay,
				"error", err,
			)
			time.Sleep(delay)
			delay *= 2
		}
	}

	return err
}

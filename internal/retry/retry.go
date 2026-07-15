package retry

import (
	"log/slog"
	"time"
)

// WithRetry executes the given function up to maxAttempts times,
// applying an exponential backoff (1s, 2s, 4s...) between attempts.
// It returns the number of attempts made and the last error encountered.
func WithRetry(fn func() error, maxAttempts int, logger *slog.Logger) (int, error) {
	var err error
	delay := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return attempt, nil // Success
		}

		if attempt < maxAttempts {
			logger.Warn("operation failed, retrying",
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"error", err,
				"next_wait", delay.String(),
			)
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		} else {
			logger.Error("operation failed, exhausted all retries",
				"attempt", attempt,
				"max_attempts", maxAttempts,
				"error", err,
			)
		}
	}

	return maxAttempts, err
}

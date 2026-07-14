package retry

import (
	"errors"
	"log/slog"
	"os"
	"testing"
)

func TestWithRetry_SuccessOnFirstAttempt(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calls := 0
	fn := func() error {
		calls++
		return nil
	}

	attempts, err := WithRetry(fn, 3, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if attempts != 1 {
		t.Fatalf("expected attempts to be 1, got %d", attempts)
	}
}

func TestWithRetry_SuccessAfterRetries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calls := 0
	fn := func() error {
		calls++
		if calls < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	// We override sleep for testing to avoid waiting
	attempts, err := WithRetry(fn, 3, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	if attempts != 3 {
		t.Fatalf("expected attempts to be 3, got %d", attempts)
	}
}

func TestWithRetry_ExhaustsAllRetries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calls := 0
	expectedErr := errors.New("fatal error")
	fn := func() error {
		calls++
		return expectedErr
	}

	attempts, err := WithRetry(fn, 3, logger)
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	if attempts != 3 {
		t.Fatalf("expected attempts to be 3, got %d", attempts)
	}
}

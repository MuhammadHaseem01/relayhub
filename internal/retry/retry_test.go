package retry

import (
	"errors"
	"testing"
)

func TestWithRetry_SuccessOnThirdAttempt(t *testing.T) {
	attempts := 0
	expectedError := errors.New("temporary error")

	fn := func() error {
		attempts++
		if attempts < 3 {
			return expectedError
		}
		return nil
	}

	err := WithRetry(fn, 3)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_FailureExhaustsAttempts(t *testing.T) {
	attempts := 0
	expectedError := errors.New("permanent error")

	fn := func() error {
		attempts++
		return expectedError
	}

	err := WithRetry(fn, 3)

	if err != expectedError {
		t.Errorf("expected %v, got %v", expectedError, err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

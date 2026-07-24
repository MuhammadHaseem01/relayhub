package webhook_test

import (
	"encoding/hex"
	"testing"

	"relayhub/internal/webhook"
)

func TestSign_Verify_RoundTrip(t *testing.T) {
	secret := []byte("super-secret-key-32-bytes-long!!")
	body := []byte(`{"event":"notification.delivered","request_id":"abc-123"}`)

	sig := webhook.Sign(secret, body)
	if sig == "" {
		t.Fatal("Sign returned empty string")
	}
	if !webhook.Verify(secret, body, sig) {
		t.Error("Verify returned false for a valid signature")
	}
}

func TestVerify_TamperedBody(t *testing.T) {
	secret := []byte("super-secret-key-32-bytes-long!!")
	body := []byte(`{"event":"notification.delivered","request_id":"abc-123"}`)
	tampered := []byte(`{"event":"notification.delivered","request_id":"EVIL-999"}`)

	sig := webhook.Sign(secret, body)
	if webhook.Verify(secret, tampered, sig) {
		t.Error("Verify returned true for a tampered body — signature check is broken")
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	secret := []byte("super-secret-key-32-bytes-long!!")
	body := []byte(`{"event":"notification.delivered","request_id":"abc-123"}`)

	wrongSig := "sha256=" + hex.EncodeToString(make([]byte, 32)) // all-zeros digest
	if webhook.Verify(secret, body, wrongSig) {
		t.Error("Verify returned true for a wrong signature — signature check is broken")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	secret := []byte("correct-secret")
	wrongSecret := []byte("wrong-secret---")
	body := []byte(`{"event":"notification.failed","request_id":"xyz-456"}`)

	sig := webhook.Sign(secret, body)
	if webhook.Verify(wrongSecret, body, sig) {
		t.Error("Verify returned true for a wrong secret")
	}
}

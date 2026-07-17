package auth

import "testing"

func TestCreateAndVerifyToken(t *testing.T) {
	token := CreateToken(42, "test-secret")
	payload, ok := VerifyToken(token, "test-secret")
	if !ok {
		t.Fatal("expected token to verify")
	}
	if payload.Subject != 42 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestVerifyTokenRejectsWrongSecret(t *testing.T) {
	token := CreateToken(42, "test-secret")
	if _, ok := VerifyToken(token, "wrong-secret"); ok {
		t.Fatal("expected token verification to fail")
	}
}

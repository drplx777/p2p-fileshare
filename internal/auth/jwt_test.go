package auth

import (
	"testing"
)

func TestNewToken_ParseToken_Roundtrip(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret")
	userID := "user-123"
	token, err := NewToken(secret, userID, 0)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	got, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if got != userID {
		t.Errorf("expected user id %q, got %q", userID, got)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret")
	if _, err := ParseToken(secret, "invalid.jwt.here"); err == nil {
		t.Error("expected error for invalid token")
	}
	if _, err := ParseToken(secret, ""); err == nil {
		t.Error("expected error for empty token")
	}
}

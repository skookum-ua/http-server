package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT_Success(t *testing.T) {
	secret := "my-super-secret-key-12345"
	userID := uuid.New()
	duration := 15 * time.Minute

	tokenStr, err := MakeJWT(userID, secret, duration)
	if err != nil {
		t.Fatalf("MakeJWT failed: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("MakeJWT returned an empty token string")
	}

	parsedID, err := ValidateJWT(tokenStr, secret)
	if err != nil {
		t.Fatalf("ValidateJWT failed on valid token: %v", err)
	}

	if parsedID != userID {
		t.Errorf("Expected UUID %v, got %v", userID, parsedID)
	}
}

func TestValidateJWT_InvalidSecret(t *testing.T) {
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"
	userID := uuid.New()

	tokenStr, _ := MakeJWT(userID, correctSecret, 15*time.Minute)

	_, err := ValidateJWT(tokenStr, wrongSecret)
	if err == nil {
		t.Error("Expected error when validating with wrong secret, but got none")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	secret := "secret"
	userID := uuid.New()
	duration := -time.Minute

	tokenStr, _ := MakeJWT(userID, secret, duration)

	_, err := ValidateJWT(tokenStr, secret)
	if err == nil {
		t.Error("Expected error for expired token, but got none")
	}
}

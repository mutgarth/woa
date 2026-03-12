package jwt_test

import (
	"testing"

	"github.com/google/uuid"
	jwtadapter "github.com/lucasmeneses/world-of-agents/server/internal/adapters/jwt"
)

func TestTokenRoundtrip(t *testing.T) {
	svc := jwtadapter.NewTokenService("test-secret")
	userID := uuid.New()
	token, err := svc.Generate(userID, "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.Validate(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != userID {
		t.Errorf("got user ID %s, want %s", claims.UserID, userID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("got email %s, want test@example.com", claims.Email)
	}
}

func TestTokenInvalid(t *testing.T) {
	svc := jwtadapter.NewTokenService("test-secret")
	_, err := svc.Validate("garbage")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

package net_test

import (
	"testing"
	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

func TestJWT_RoundTrip(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	auth := wonet.NewAuth(secret)
	token, err := auth.GenerateJWT("user-123", "lucas@test.com")
	if err != nil { t.Fatalf("generate JWT: %v", err) }
	claims, err := auth.ValidateJWT(token)
	if err != nil { t.Fatalf("validate JWT: %v", err) }
	if claims.UserID != "user-123" { t.Fatalf("expected user-123, got %s", claims.UserID) }
	if claims.Email != "lucas@test.com" { t.Fatalf("expected lucas@test.com, got %s", claims.Email) }
}

func TestJWT_InvalidToken(t *testing.T) {
	auth := wonet.NewAuth("test-secret-key-32-bytes-long!!")
	_, err := auth.ValidateJWT("invalid.token.here")
	if err == nil { t.Fatal("expected error for invalid token") }
}

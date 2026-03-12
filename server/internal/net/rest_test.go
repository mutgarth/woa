package net_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wonet "github.com/lucasmeneses/world-of-agents/server/internal/net"
)

func TestStatsEndpoint(t *testing.T) {
	rest := wonet.NewREST(nil, nil, nil) // stats doesn't need auth
	mux := http.NewServeMux()
	rest.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "online") {
		t.Fatal("expected 'online' in response")
	}
}

func TestBadJSON(t *testing.T) {
	rest := wonet.NewREST(nil, nil, nil) // register will fail on nil auth, but bad JSON is caught first
	mux := http.NewServeMux()
	rest.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/auth/register", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

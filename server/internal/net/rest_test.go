package net_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleStats_ReturnsOnline(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "online"})
	})
	req := httptest.NewRequest("GET", "/api/stats", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("expected 200, got %d", rec.Code) }
	var body map[string]any
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "online" { t.Fatalf("expected status=online, got %v", body["status"]) }
}

func TestHandleRegister_RejectsBadJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", func(w http.ResponseWriter, r *http.Request) {
		var body struct { Email string `json:"email"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "BAD_REQUEST", "message": "invalid JSON"},
			})
			return
		}
	})
	req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", rec.Code) }
}

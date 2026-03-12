// server/internal/net/helpers.go
package net

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const ctxKeyUserID contextKey = "user_id"

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func userIDFromContext(r *http.Request) uuid.UUID {
	return r.Context().Value(ctxKeyUserID).(uuid.UUID)
}

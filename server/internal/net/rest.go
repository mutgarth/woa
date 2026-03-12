// server/internal/net/rest.go
package net

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/agent"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/auth"
)

type REST struct {
	auth *auth.Service
}

func NewREST(authService *auth.Service) *REST {
	return &REST{auth: authService}
}

func (r *REST) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", r.handleRegister)
	mux.HandleFunc("POST /auth/login", r.handleLogin)
	mux.HandleFunc("GET /api/agents", r.requireAuth(r.handleListAgents))
	mux.HandleFunc("POST /api/agents", r.requireAuth(r.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{id}", r.requireAuth(r.handleDeleteAgent))
	mux.HandleFunc("GET /api/stats", r.handleStats)
}

func (r *REST) handleRegister(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if body.Email == "" || body.Password == "" || body.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "email, password, and display_name required")
		return
	}
	user, token, err := r.auth.Register(req.Context(), body.Email, body.Password, body.DisplayName)
	if err != nil {
		writeError(w, http.StatusConflict, "EMAIL_TAKEN", "email already registered")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": user.ID.String(), "token": token})
}

func (r *REST) handleLogin(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	user, token, err := r.auth.Login(req.Context(), body.Email, body.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": user.ID.String(), "token": token})
}

func (r *REST) handleCreateAgent(w http.ResponseWriter, req *http.Request) {
	userID := userIDFromContext(req)
	var body struct {
		Name      string `json:"name"`
		AgentType string `json:"agent_type"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON")
		return
	}
	if body.Name == "" || body.AgentType == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "name and agent_type required")
		return
	}
	a, apiKey, err := r.auth.CreateAgent(req.Context(), userID, body.Name, agent.AgentType(body.AgentType))
	if err != nil {
		writeError(w, http.StatusConflict, "AGENT_EXISTS", "agent name already taken")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"agent_id": a.ID.String(), "name": a.Name,
		"agent_type": string(a.AgentType), "api_key": apiKey,
	})
}

func (r *REST) handleListAgents(w http.ResponseWriter, req *http.Request) {
	userID := userIDFromContext(req)
	agents, err := r.auth.ListAgents(req.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list agents")
		return
	}
	result := make([]map[string]any, len(agents))
	for i, a := range agents {
		result[i] = map[string]any{"id": a.ID.String(), "name": a.Name, "agent_type": string(a.AgentType)}
	}
	writeJSON(w, http.StatusOK, result)
}

func (r *REST) handleDeleteAgent(w http.ResponseWriter, req *http.Request) {
	userID := userIDFromContext(req)
	agentID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid agent ID")
		return
	}
	if err := r.auth.DeleteAgent(req.Context(), agentID, userID); err != nil {
		writeError(w, http.StatusNotFound, "AGENT_NOT_FOUND", "agent not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *REST) handleStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "online"})
}

func (r *REST) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "missing Authorization header")
			return
		}
		claims, err := r.auth.AuthenticateByToken(req.Context(), authHeader[7:])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "AUTH_FAILED", "invalid token")
			return
		}
		ctx := context.WithValue(req.Context(), ctxKeyUserID, claims.UserID)
		next(w, req.WithContext(ctx))
	}
}

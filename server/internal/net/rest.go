// server/internal/net/rest.go
package net

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/agent"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/auth"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/guild"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/task"
)

type REST struct {
	auth   *auth.Service
	guilds *guild.Service
	tasks  *task.Service
}

func NewREST(authService *auth.Service, guildService *guild.Service, taskService *task.Service) *REST {
	return &REST{auth: authService, guilds: guildService, tasks: taskService}
}

func (r *REST) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", r.handleRegister)
	mux.HandleFunc("POST /auth/login", r.handleLogin)
	mux.HandleFunc("GET /api/agents", r.requireAuth(r.handleListAgents))
	mux.HandleFunc("POST /api/agents", r.requireAuth(r.handleCreateAgent))
	mux.HandleFunc("DELETE /api/agents/{id}", r.requireAuth(r.handleDeleteAgent))
	mux.HandleFunc("GET /api/stats", r.handleStats)
	mux.HandleFunc("GET /api/guilds", r.requireAuth(r.handleListGuilds))
	mux.HandleFunc("GET /api/guilds/{id}", r.requireAuth(r.handleGetGuild))
	mux.HandleFunc("GET /api/guilds/{id}/tasks", r.requireAuth(r.handleListGuildTasks))
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

func (r *REST) handleListGuilds(w http.ResponseWriter, req *http.Request) {
	limit, offset := parsePagination(req)
	guilds, err := r.guilds.List(req.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list guilds")
		return
	}
	result := make([]map[string]any, len(guilds))
	for i, g := range guilds {
		result[i] = map[string]any{
			"id": g.ID.String(), "name": g.Name,
			"description": g.Description, "visibility": string(g.Visibility),
			"max_members": g.MaxMembers, "created_at": g.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (r *REST) handleGetGuild(w http.ResponseWriter, req *http.Request) {
	guildID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid guild ID")
		return
	}
	g, members, err := r.guilds.GetWithMembers(req.Context(), guildID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "guild not found")
		return
	}
	memberList := make([]map[string]any, len(members))
	for i, m := range members {
		memberList[i] = map[string]any{
			"agent_id": m.AgentID.String(), "role": string(m.Role), "joined_at": m.JoinedAt,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": g.ID.String(), "name": g.Name, "description": g.Description,
		"visibility": string(g.Visibility), "max_members": g.MaxMembers,
		"created_at": g.CreatedAt, "members": memberList,
	})
}

func (r *REST) handleListGuildTasks(w http.ResponseWriter, req *http.Request) {
	guildID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid guild ID")
		return
	}
	limit, offset := parsePagination(req)

	var statusFilter *task.Status
	if s := req.URL.Query().Get("status"); s != "" {
		status := task.Status(s)
		statusFilter = &status
	}

	tasks, err := r.tasks.List(req.Context(), guildID, statusFilter, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tasks")
		return
	}
	result := make([]map[string]any, len(tasks))
	for i, t := range tasks {
		entry := map[string]any{
			"id": t.ID.String(), "guild_id": t.GuildID.String(),
			"posted_by": t.PostedBy.String(), "title": t.Title,
			"description": t.Description, "priority": string(t.Priority),
			"status": string(t.Status), "created_at": t.CreatedAt,
		}
		if t.ClaimedBy != nil {
			entry["claimed_by"] = t.ClaimedBy.String()
		}
		if t.Result != "" {
			entry["result"] = t.Result
		}
		result[i] = entry
	}
	writeJSON(w, http.StatusOK, result)
}

func parsePagination(req *http.Request) (int, int) {
	limit := 50
	offset := 0
	if l := req.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := req.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

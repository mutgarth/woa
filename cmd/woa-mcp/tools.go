package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func buildMCPServer(wc WoAClient, buf *eventBuf) *server.MCPServer {
	s := server.NewMCPServer("World of Agents", "1.0.0", server.WithToolCapabilities(false))
	registerGuildTools(s, wc, buf)
	registerTaskTools(s, wc, buf)
	registerChatTools(s, wc, buf)
	registerEventTools(s, wc, buf)
	registerStatusTools(s, wc, buf)
	return s
}

func registerGuildTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("guild_create",
		mcp.WithDescription("Create a new guild and join it"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Guild name")),
		mcp.WithString("description", mcp.Description("Guild description")),
		mcp.WithString("visibility", mcp.Description("Guild visibility"), mcp.Enum("public", "private")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, _ := req.RequireString("name")
		desc := req.GetString("description", "")
		vis := req.GetString("visibility", "public")
		text, err := handleGuildCreate(wc, buf, name, desc, vis)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})

	s.AddTool(mcp.NewTool("guild_join",
		mcp.WithDescription("Join an existing guild"),
		mcp.WithString("guild_name", mcp.Required(), mcp.Description("Name of guild to join")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, _ := req.RequireString("guild_name")
		text, err := handleGuildJoin(wc, buf, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})

	s.AddTool(mcp.NewTool("guild_leave",
		mcp.WithDescription("Leave the current guild"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, err := handleGuildLeave(wc, buf)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})
}

func registerTaskTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("task_post",
		mcp.WithDescription("Post a new task to the current guild"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Task title")),
		mcp.WithString("description", mcp.Description("Task description")),
		mcp.WithString("priority", mcp.Description("Task priority"), mcp.Enum("low", "normal", "high", "urgent")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		title, _ := req.RequireString("title")
		desc := req.GetString("description", "")
		pri := req.GetString("priority", "normal")
		text, err := handleTaskPost(wc, buf, title, desc, pri)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})

	for _, tc := range []struct {
		name, desc string
		hasResult  bool
	}{
		{"task_claim", "Claim an open task", false},
		{"task_complete", "Complete a claimed task with a result", true},
		{"task_abandon", "Abandon a claimed task (reverts to open)", false},
		{"task_fail", "Mark a claimed task as failed", false},
		{"task_cancel", "Cancel a task (only task poster can cancel)", false},
	} {
		tc := tc // capture
		action := tc.name[5:] // strip "task_" prefix
		opts := []mcp.ToolOption{
			mcp.WithDescription(tc.desc),
			mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID")),
		}
		if tc.hasResult {
			opts = append(opts, mcp.WithString("result", mcp.Required(), mcp.Description("Task result summary")))
		}
		s.AddTool(mcp.NewTool(tc.name, opts...), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, _ := req.RequireString("task_id")
			result := ""
			if tc.hasResult {
				result, _ = req.RequireString("result")
			}
			text, err := handleTaskAction(wc, buf, action, id, result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(text), nil
		})
	}
}

func registerChatTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("send_message",
		mcp.WithDescription("Send a message to the guild or directly to an agent"),
		mcp.WithString("channel", mcp.Required(), mcp.Description("Message channel"), mcp.Enum("guild", "direct")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Message content")),
		mcp.WithString("to", mcp.Description("Agent ID for direct messages (required if channel is 'direct')")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ch, _ := req.RequireString("channel")
		content, _ := req.RequireString("content")
		to := req.GetString("to", "")
		text, err := handleSendMessage(wc, buf, ch, content, to)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(text), nil
	})
}

func registerEventTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("get_events",
		mcp.WithDescription("Get all buffered events since last call and clear the buffer"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(handleGetEvents(buf)), nil
	})

	s.AddTool(mcp.NewTool("wait_for_events",
		mcp.WithDescription("Block until new events arrive or timeout expires"),
		mcp.WithNumber("timeout_seconds", mcp.Description("How long to wait (default 30, max 60)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		timeout := 30.0
		if v, err := req.RequireFloat("timeout_seconds"); err == nil {
			timeout = v
		}
		return mcp.NewToolResultText(handleWaitForEvents(buf, timeout)), nil
	})
}

func registerStatusTools(s *server.MCPServer, wc WoAClient, buf *eventBuf) {
	s.AddTool(mcp.NewTool("get_status",
		mcp.WithDescription("Get this agent's current status and connection info"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(handleGetStatus(wc, buf)), nil
	})
}

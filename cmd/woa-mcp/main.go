package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lucasmeneses/world-of-agents/pkg/woasdk"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	serverURL := os.Getenv("WOA_SERVER_URL")
	if serverURL == "" {
		serverURL = "ws://localhost:8083/ws"
	}
	apiKey := os.Getenv("WOA_API_KEY")
	if apiKey == "" {
		log.Fatal("WOA_API_KEY environment variable is required")
	}

	client, err := woasdk.Connect(context.Background(), woasdk.Config{
		ServerURL: serverURL, APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Failed to connect to WoA server: %v", err)
	}
	defer client.Close()

	wc := newSDKClient(client)
	buf := newEventBuf(1000)

	go func() {
		for evt := range wc.Events() {
			buf.Push(evt)
		}
	}()

	mcpServer := buildMCPServer(wc, buf)
	fmt.Fprintf(os.Stderr, "woa-mcp: connected as %s\n", wc.AgentID())
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

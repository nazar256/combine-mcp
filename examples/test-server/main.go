package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithLogging(),
	)

	// Add a simple echo tool
	s.AddTool(mcp.NewTool("echo",
		mcp.WithDescription("Echo back the input message"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Message to echo back"),
		),
	), echoHandler)

	// Add a simple addition tool
	s.AddTool(mcp.NewTool("add",
		mcp.WithDescription("Add two numbers"),
		mcp.WithNumber("a",
			mcp.Required(),
			mcp.Description("First number"),
		),
		mcp.WithNumber("b",
			mcp.Required(),
			mcp.Description("Second number"),
		),
	), addHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func echoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, ok := request.Params.Arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message must be a string")
	}

	return mcp.NewToolResultText(fmt.Sprintf("Echo: %s", message)), nil
}

func addHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a, ok := request.Params.Arguments["a"].(float64)
	if !ok {
		return nil, fmt.Errorf("a must be a number")
	}

	b, ok := request.Params.Arguments["b"].(float64)
	if !ok {
		return nil, fmt.Errorf("b must be a number")
	}

	result := a + b
	return mcp.NewToolResultText(fmt.Sprintf("Result: %f", result)), nil
}

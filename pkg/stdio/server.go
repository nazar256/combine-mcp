package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nazar256/combine-mcp/pkg/aggregator"
	"github.com/nazar256/combine-mcp/pkg/logger"
)

// AggregatorServer represents the MCP server that aggregates tools from multiple MCP servers
type AggregatorServer struct {
	mcpServer  *server.MCPServer
	aggregator *aggregator.MCPAggregator
}

// NewAggregatorServer creates a new AggregatorServer
func NewAggregatorServer(serverName, version string, aggregator *aggregator.MCPAggregator) *AggregatorServer {
	// Add debug hooks
	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(id any, method mcp.MCPMethod, message any) {
		logger.Debug("Before method: %s, id: %v", method, id)
	})

	hooks.AddOnSuccess(func(id any, method mcp.MCPMethod, message any, result any) {
		logger.Debug("Success method: %s, id: %v", method, id)
	})

	hooks.AddOnError(func(id any, method mcp.MCPMethod, message any, err error) {
		logger.Error("Error in method: %s, id: %v, error: %v", method, id, err)
	})

	hooks.AddBeforeInitialize(func(id any, message *mcp.InitializeRequest) {
		logger.Info("Initialize request from: %s %s", message.Params.ClientInfo.Name, message.Params.ClientInfo.Version)
		logger.Debug("Initialize params: %+v", message.Params)

		// Check if we have a custom protocol version to use (for compatibility)
		if protocolVersion := os.Getenv("MCP_PROTOCOL_VERSION"); protocolVersion != "" {
			logger.Info("Overriding protocol version to %s for compatibility", protocolVersion)
			message.Params.ProtocolVersion = protocolVersion
		}
	})

	hooks.AddAfterInitialize(func(id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		logger.Info("Initialize response: server %s %s", result.ServerInfo.Name, result.ServerInfo.Version)

		// Check if we're in Cursor mode
		if os.Getenv("MCP_CURSOR_MODE") != "" {
			logger.Info("Cursor compatibility mode enabled - customizing response")

			// Cursor might expect a specific server name format
			if result.ServerInfo.Name != "cursor-mcp-server" {
				logger.Debug("Setting server name to cursor-mcp-server for compatibility")
				result.ServerInfo.Name = "cursor-mcp-server"
			}
		}
	})

	hooks.AddBeforeCallTool(func(id any, message *mcp.CallToolRequest) {
		logger.Info("Tool call: %s, id: %v", message.Params.Name, id)
		logger.Debug("Tool arguments: %+v", message.Params.Arguments)
	})

	hooks.AddAfterCallTool(func(id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		logger.Info("Tool call result: %s, success: %v", message.Params.Name, !result.IsError)
	})

	mcpServer := server.NewMCPServer(
		serverName,
		version,
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	return &AggregatorServer{
		mcpServer:  mcpServer,
		aggregator: aggregator,
	}
}

// RegisterTools registers all tools from the aggregator to the MCP server
func (s *AggregatorServer) RegisterTools() error {
	// Get tools from aggregator
	tools := s.aggregator.GetTools()
	logger.Info("Registering %d tools from aggregator", len(tools))

	// Register each tool with the MCP server
	for _, tool := range tools {
		logger.Debug("Registering tool: %s", tool.Name)
		s.mcpServer.AddTool(
			mcp.Tool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			},
			s.createToolHandler(tool.Name),
		)
	}

	return nil
}

// createToolHandler creates a handler function for a specific tool
func (s *AggregatorServer) createToolHandler(toolName string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Forward the call to the aggregator
		logger.Debug("Handling tool call: %s", toolName)
		result, err := s.aggregator.CallTool(ctx, request)
		if err != nil {
			logger.Error("Tool call failed: %s, error: %v", toolName, err)
		} else {
			logger.Debug("Tool call succeeded: %s", toolName)
		}
		return result, err
	}
}

// ServeStdio serves the MCP server over stdio with message logging
func (s *AggregatorServer) ServeStdio() error {
	logger.Debug("Starting stdio server")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // Increase scanner buffer size
	ctx := context.Background()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		// Log incoming message to file only with extra detail
		logger.LogRPC("IN", line)

		// Try to parse the incoming message for better logging
		var req map[string]interface{}
		if err := json.Unmarshal(line, &req); err == nil {
			if method, ok := req["method"].(string); ok {
				id := "null"
				if reqID, exists := req["id"]; exists {
					id = fmt.Sprintf("%v", reqID)
				}
				logger.Debug("Received request: method=%s, id=%s", method, id)
			}
		}

		// Handle message
		response := s.mcpServer.HandleMessage(ctx, line)
		if response != nil {
			responseBytes, err := json.Marshal(response)
			if err != nil {
				logger.Error("Failed to marshal response: %v", err)
				continue
			}

			// Log outgoing message to file only with extra detail
			logger.LogRPC("OUT", responseBytes)

			// Try to parse the response for better logging
			var resp map[string]interface{}
			if err := json.Unmarshal(responseBytes, &resp); err == nil {
				id := "null"
				if respID, exists := resp["id"]; exists {
					id = fmt.Sprintf("%v", respID)
				}

				if result, exists := resp["result"]; exists {
					logger.Debug("Sending response: id=%s, success=true", id)

					// For tools/list specifically, log the count of tools
					if toolsResult, ok := result.(map[string]interface{}); ok {
						if tools, exists := toolsResult["tools"].([]interface{}); exists {
							logger.Debug("Response includes %d tools", len(tools))
						}
					}
				} else if _, exists := resp["error"]; exists {
					logger.Debug("Sending response: id=%s, error=true", id)
				}
			}

			// Write response - this must be the only thing written to stdout
			// No logging, no extra output, just the pure JSON response
			// We explicitly use os.Stdout to ensure we're writing to the original stdout
			fmt.Fprintln(os.Stdout, string(responseBytes))
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Scanner error: %v", err)
		return err
	}

	return nil
}

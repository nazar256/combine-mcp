package aggregator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/yn/combine-mcp/pkg/config"
	"github.com/yn/combine-mcp/pkg/logger"
)

// MCPAggregator is responsible for aggregating multiple MCP servers
type MCPAggregator struct {
	clients map[string]*client.StdioMCPClient
	tools   map[string]toolMapping
	mu      sync.RWMutex
}

type toolMapping struct {
	serverName    string
	originalName  string
	sanitizedName string
}

// sanitizeToolName replaces dashes with underscores in a tool name to make it compatible with Cursor
func sanitizeToolName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// NewMCPAggregator creates a new MCPAggregator
func NewMCPAggregator() *MCPAggregator {
	return &MCPAggregator{
		clients: make(map[string]*client.StdioMCPClient),
		tools:   make(map[string]toolMapping),
	}
}

// Initialize initializes connections to all configured MCP servers
func (a *MCPAggregator) Initialize(ctx context.Context, cfg *config.Config) error {
	// Override the os.Stdout during initialization to redirect it to stderr
	// This prevents any subprocess output from corrupting our JSON stdout
	oldStdout := os.Stdout

	// Create a pipe to capture any potential stdout output from subprocesses
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create output pipe: %w", err)
	}

	// Replace stdout with our pipe temporarily
	os.Stdout = w

	// Start a goroutine to read from the pipe and write to stderr
	go func() {
		defer r.Close()
		buffer := make([]byte, 1024)
		for {
			n, err := r.Read(buffer)
			if err != nil {
				if err != os.ErrClosed {
					logger.Error("Error reading subprocess output: %v", err)
				}
				break
			}
			if n > 0 {
				// Write subprocess output to stderr instead of stdout
				os.Stderr.Write(buffer[:n])
			}
		}
	}()

	// Make sure we restore stdout when we're done
	defer func() {
		w.Close()
		os.Stdout = oldStdout
	}()

	for _, serverCfg := range cfg.Servers {
		// Convert environment variables to string array format
		var envVars []string
		for key, value := range serverCfg.Env {
			envVars = append(envVars, key+"="+value)
		}

		// Debug output to file only
		logger.Debug("Initializing MCP server %s with command: %s %v", serverCfg.Name, serverCfg.Command, serverCfg.Args)
		logger.Debug("Environment variables: %v", envVars)

		// Create an exec.Cmd manually to control stderr redirection
		cmd := exec.Command(serverCfg.Command, serverCfg.Args...)
		cmd.Stderr = os.Stderr // Redirect stderr to stderr
		cmd.Env = append(os.Environ(), envVars...)

		// Create client
		mcpClient, err := client.NewStdioMCPClient(
			serverCfg.Command,
			envVars,
			serverCfg.Args...,
		)
		if err != nil {
			logger.Error("Failed to create client for server %s: %v", serverCfg.Name, err)
			return fmt.Errorf("failed to create client for server %s: %w", serverCfg.Name, err)
		}

		// Initialize the client with longer timeout for NPM packages
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		// Initialize the client
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "mcp-aggregator",
			Version: "1.0.0",
		}

		logger.Debug("Sending initialize request to %s...", serverCfg.Name)
		initResult, err := mcpClient.Initialize(ctxWithTimeout, initRequest)
		if err != nil {
			mcpClient.Close()
			logger.Error("Failed to initialize server %s: %v", serverCfg.Name, err)

			// Check if this is a context cancellation or deadline exceeded error
			// We want to handle these more gracefully
			if ctxWithTimeout.Err() != nil || strings.Contains(err.Error(), "context") {
				logger.Error("Context error for server %s: %v", serverCfg.Name, err)
				logger.Error("Skipping server %s", serverCfg.Name)
				continue // Skip this server but continue with others
			}

			// For other errors, we'll continue with other servers but log the error
			logger.Error("Error initializing server %s: %v", serverCfg.Name, err)
			logger.Error("Continuing with other servers...")
			continue
		}
		logger.Info("Server %s initialized: %s %s", serverCfg.Name, initResult.ServerInfo.Name, initResult.ServerInfo.Version)

		// Store the client
		a.mu.Lock()
		a.clients[serverCfg.Name] = mcpClient
		a.mu.Unlock()

		// Discover tools and register them with prefix
		err = a.discoverTools(ctx, serverCfg.Name)
		if err != nil {
			logger.Error("Failed to discover tools for server %s: %v", serverCfg.Name, err)
			// Continue with other servers even if tool discovery fails
			logger.Error("Continuing with other servers...")
			continue
		}
	}

	// Check if we have at least one server initialized
	if len(a.clients) == 0 {
		return fmt.Errorf("no servers were successfully initialized")
	}

	return nil
}

// discoverTools discovers all tools available on a server and registers them with a prefix
func (a *MCPAggregator) discoverTools(ctx context.Context, serverName string) error {
	a.mu.RLock()
	mcpClient, exists := a.clients[serverName]
	a.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client for server %s not found", serverName)
	}

	// Get tools using list method
	logger.Debug("Discovering tools for server %s...", serverName)
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list tools for server %s: %w", serverName, err)
	}
	logger.Debug("Found %d tools for server %s", len(toolsResp.Tools), serverName)

	// Register each tool with a prefix
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, tool := range toolsResp.Tools {
		originalName := tool.Name
		sanitizedName := sanitizeToolName(originalName)
		prefixedName := fmt.Sprintf("%s_%s", serverName, sanitizedName)

		logger.Debug("Registering tool: %s -> %s (sanitized from: %s)", originalName, prefixedName, tool.Name)

		a.tools[prefixedName] = toolMapping{
			serverName:    serverName,
			originalName:  originalName,
			sanitizedName: sanitizedName,
		}
	}

	return nil
}

// GetTools returns a list of all tools from all servers with prefixed names
func (a *MCPAggregator) GetTools() []mcp.Tool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get tools from all servers
	var allTools []mcp.Tool
	for prefixedName, mapping := range a.tools {
		mcpClient := a.clients[mapping.serverName]

		// Get the original tool schema using ListTools
		toolsResp, err := mcpClient.ListTools(context.Background(), mcp.ListToolsRequest{})
		if err != nil {
			// Skip tools that can't be retrieved
			logger.Error("Error getting tools for %s: %v", mapping.serverName, err)
			continue
		}

		// Find the specific tool
		var tool mcp.Tool
		found := false
		for _, t := range toolsResp.Tools {
			if t.Name == mapping.originalName {
				tool = t
				found = true
				break
			}
		}

		if !found {
			logger.Debug("Tool %s not found in server %s", mapping.originalName, mapping.serverName)
			continue
		}

		// Create a new tool with the prefixed name (with underscores instead of dashes)
		tool.Name = prefixedName

		// Update the description to indicate the source server
		if tool.Description != "" {
			tool.Description = fmt.Sprintf("[%s] %s", mapping.serverName, tool.Description)
		}

		// Ensure the tool has a valid input schema for Cursor
		ensureValidToolSchema(&tool)

		allTools = append(allTools, tool)
	}

	return allTools
}

// ensureValidToolSchema ensures the tool's input schema is in a format Cursor expects
func ensureValidToolSchema(tool *mcp.Tool) {
	// Ensure the input schema has required fields
	if tool.InputSchema.Type == "" {
		tool.InputSchema.Type = "object"
	}

	// Ensure properties field exists and is initialized
	if tool.InputSchema.Properties == nil {
		tool.InputSchema.Properties = make(map[string]interface{})
	}
}

// CallTool calls a tool on the appropriate server
func (a *MCPAggregator) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a.mu.RLock()
	prefixedName := request.Params.Name
	mapping, exists := a.tools[prefixedName]
	mcpClient, clientExists := a.clients[mapping.serverName]
	a.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool %s not found", prefixedName)
	}

	if !clientExists {
		return nil, fmt.Errorf("client for server %s not found", mapping.serverName)
	}

	logger.Debug("Calling tool %s on server %s (mapped from %s)", mapping.originalName, mapping.serverName, prefixedName)

	// Create a new request with the original tool name (without prefix and with original dashes)
	newRequest := request
	newRequest.Params.Name = mapping.originalName

	// Call the tool on the appropriate server
	return mcpClient.CallTool(ctx, newRequest)
}

// Close closes all client connections
func (a *MCPAggregator) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for name, mcpClient := range a.clients {
		mcpClient.Close()
		delete(a.clients, name)
	}
}

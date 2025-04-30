package aggregator

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nazar256/combine-mcp/pkg/config"
)

func TestSanitizeToolName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No dashes",
			input:    "getuser",
			expected: "getuser",
		},
		{
			name:     "Single dash",
			input:    "get-user",
			expected: "get_user",
		},
		{
			name:     "Multiple dashes",
			input:    "get-user-details",
			expected: "get_user_details",
		},
		{
			name:     "Already has underscores",
			input:    "get_user",
			expected: "get_user",
		},
		{
			name:     "Mixed dashes and underscores",
			input:    "get_user-details",
			expected: "get_user_details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeToolName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeToolName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// MockClient implements a simple mock for testing without real StdioMCPClient
type MockClient struct {
	Tools []mcp.Tool
}

func (m *MockClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return &mcp.InitializeResult{
		ServerInfo: mcp.Implementation{
			Name:    "mock-server",
			Version: "1.0.0",
		},
	}, nil
}

func (m *MockClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{
		Tools: m.Tools,
	}, nil
}

func (m *MockClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Just return a simple success result for testing
	return &mcp.CallToolResult{}, nil
}

func (m *MockClient) Close() error {
	return nil
}

func TestToolNameSanitization(t *testing.T) {
	// Create test tools with dashes
	testTools := []struct {
		originalName  string
		sanitizedName string
		description   string
	}{
		{
			originalName:  "get-user",
			sanitizedName: "get_user",
			description:   "Get user details",
		},
		{
			originalName:  "create-issue",
			sanitizedName: "create_issue",
			description:   "Create a new issue",
		},
	}

	// Verify sanitization
	for _, tt := range testTools {
		result := sanitizeToolName(tt.originalName)
		if result != tt.sanitizedName {
			t.Errorf("sanitizeToolName(%q) = %q, want %q", tt.originalName, result, tt.sanitizedName)
		}
	}
}

func TestToolFiltering(t *testing.T) {
	tests := []struct {
		name          string
		serverConfig  config.ServerConfig
		serverTools   []mcp.Tool
		wantToolNames []string
	}{
		{
			name: "No filtering returns all tools",
			serverConfig: config.ServerConfig{
				Name:    "test-server",
				Command: "test-command",
			},
			serverTools: []mcp.Tool{
				{Name: "tool1", Description: "Tool 1"},
				{Name: "tool2", Description: "Tool 2"},
			},
			wantToolNames: []string{"test_server_tool1", "test_server_tool2"},
		},
		{
			name: "Filtered tools returns only allowed tools",
			serverConfig: config.ServerConfig{
				Name:    "test-server",
				Command: "test-command",
				Tools: &config.ToolsConfig{
					Allowed: []string{"tool1"},
				},
			},
			serverTools: []mcp.Tool{
				{Name: "tool1", Description: "Tool 1"},
				{Name: "tool2", Description: "Tool 2"},
			},
			wantToolNames: []string{"test_server_tool1"},
		},
		{
			name: "Empty allowed list returns no tools",
			serverConfig: config.ServerConfig{
				Name:    "test-server",
				Command: "test-command",
				Tools: &config.ToolsConfig{
					Allowed: []string{},
				},
			},
			serverTools: []mcp.Tool{
				{Name: "tool1", Description: "Tool 1"},
				{Name: "tool2", Description: "Tool 2"},
			},
			wantToolNames: []string{},
		},
		{
			name: "Non-existent allowed tools are ignored",
			serverConfig: config.ServerConfig{
				Name:    "test-server",
				Command: "test-command",
				Tools: &config.ToolsConfig{
					Allowed: []string{"tool1", "non-existent"},
				},
			},
			serverTools: []mcp.Tool{
				{Name: "tool1", Description: "Tool 1"},
				{Name: "tool2", Description: "Tool 2"},
			},
			wantToolNames: []string{"test_server_tool1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client with the test tools
			mockClient := &MockClient{
				Tools: tt.serverTools,
			}

			// Create an aggregator and add the mock client
			agg := NewMCPAggregator()
			agg.clients[tt.serverConfig.Name] = mockClient
			agg.configs[tt.serverConfig.Name] = &tt.serverConfig

			// Register tools for the server
			err := agg.discoverTools(context.Background(), tt.serverConfig.Name)
			if err != nil {
				t.Fatalf("discoverTools() error = %v", err)
			}

			// Get all tools and check the results
			tools := agg.GetTools()
			gotToolNames := make([]string, 0, len(tools))
			for _, tool := range tools {
				gotToolNames = append(gotToolNames, tool.Name)
			}

			// Compare tool names
			if len(gotToolNames) != len(tt.wantToolNames) {
				t.Errorf("Got %d tools, want %d", len(gotToolNames), len(tt.wantToolNames))
				return
			}

			// Create maps for easier comparison
			gotMap := make(map[string]bool)
			wantMap := make(map[string]bool)
			for _, name := range gotToolNames {
				gotMap[name] = true
			}
			for _, name := range tt.wantToolNames {
				wantMap[name] = true
			}

			// Check for missing tools
			for name := range wantMap {
				if !gotMap[name] {
					t.Errorf("Missing tool: %s", name)
				}
			}

			// Check for extra tools
			for name := range gotMap {
				if !wantMap[name] {
					t.Errorf("Extra tool: %s", name)
				}
			}
		})
	}
}

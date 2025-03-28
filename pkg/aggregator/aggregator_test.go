package aggregator

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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

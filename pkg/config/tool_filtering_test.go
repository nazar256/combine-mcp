package config

import (
	"encoding/json"
	"testing"
)

func TestToolsConfig(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    *Config
		wantErr bool
	}{
		{
			name: "Valid config with tool filtering",
			json: `{
				"servers": [
					{
						"name": "test-server",
						"command": "test-command",
						"tools": {
							"allowed": ["tool1", "tool2"]
						}
					}
				]
			}`,
			want: &Config{
				Servers: []ServerConfig{
					{
						Name:    "test-server",
						Command: "test-command",
						Tools: &ToolsConfig{
							Allowed: []string{"tool1", "tool2"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid config without tool filtering",
			json: `{
				"servers": [
					{
						"name": "test-server",
						"command": "test-command"
					}
				]
			}`,
			want: &Config{
				Servers: []ServerConfig{
					{
						Name:    "test-server",
						Command: "test-command",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid config with empty allowed tools",
			json: `{
				"servers": [
					{
						"name": "test-server",
						"command": "test-command",
						"tools": {
							"allowed": []
						}
					}
				]
			}`,
			want: &Config{
				Servers: []ServerConfig{
					{
						Name:    "test-server",
						Command: "test-command",
						Tools: &ToolsConfig{
							Allowed: []string{},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid config with multiple servers and mixed filtering",
			json: `{
				"servers": [
					{
						"name": "server1",
						"command": "command1",
						"tools": {
							"allowed": ["tool1", "tool2"]
						}
					},
					{
						"name": "server2",
						"command": "command2"
					}
				]
			}`,
			want: &Config{
				Servers: []ServerConfig{
					{
						Name:    "server1",
						Command: "command1",
						Tools: &ToolsConfig{
							Allowed: []string{"tool1", "tool2"},
						},
					},
					{
						Name:    "server2",
						Command: "command2",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Config
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if len(got.Servers) != len(tt.want.Servers) {
					t.Errorf("Config has wrong number of servers, got %d want %d", len(got.Servers), len(tt.want.Servers))
					return
				}
				for i, server := range got.Servers {
					wantServer := tt.want.Servers[i]
					if server.Name != wantServer.Name {
						t.Errorf("Server[%d].Name = %v, want %v", i, server.Name, wantServer.Name)
					}
					if server.Command != wantServer.Command {
						t.Errorf("Server[%d].Command = %v, want %v", i, server.Command, wantServer.Command)
					}
					if (server.Tools == nil) != (wantServer.Tools == nil) {
						t.Errorf("Server[%d].Tools presence mismatch, got %v, want %v", i, server.Tools != nil, wantServer.Tools != nil)
						continue
					}
					if server.Tools != nil {
						if len(server.Tools.Allowed) != len(wantServer.Tools.Allowed) {
							t.Errorf("Server[%d].Tools.Allowed length = %d, want %d", i, len(server.Tools.Allowed), len(wantServer.Tools.Allowed))
							continue
						}
						for j, tool := range server.Tools.Allowed {
							if tool != wantServer.Tools.Allowed[j] {
								t.Errorf("Server[%d].Tools.Allowed[%d] = %v, want %v", i, j, tool, wantServer.Tools.Allowed[j])
							}
						}
					}
				}
			}
		})
	}
}

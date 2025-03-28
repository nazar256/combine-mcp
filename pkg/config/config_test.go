package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	validConfig := Config{
		Servers: []ServerConfig{
			{
				Name:    "test-server",
				Command: "/path/to/test-server",
				Args:    []string{"--arg1", "--arg2"},
			},
		},
		// Set expected LogLevel to match default (what GetLogLevel returns)
		LogLevel: LogLevelInfo,
	}

	validConfigJSON, err := json.Marshal(validConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	tempDir := t.TempDir()
	validConfigPath := filepath.Join(tempDir, "valid-config.json")
	if err := os.WriteFile(validConfigPath, validConfigJSON, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create an invalid JSON config file
	invalidJSONPath := filepath.Join(tempDir, "invalid-json.json")
	if err := os.WriteFile(invalidJSONPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	// Create an empty config file
	emptyConfigJSON, err := json.Marshal(Config{})
	if err != nil {
		t.Fatalf("Failed to marshal empty config: %v", err)
	}
	emptyConfigPath := filepath.Join(tempDir, "empty-config.json")
	if err := os.WriteFile(emptyConfigPath, emptyConfigJSON, 0644); err != nil {
		t.Fatalf("Failed to write empty config file: %v", err)
	}

	// Test cases
	tests := []struct {
		name       string
		envVar     string
		envValue   string
		wantConfig *Config
		wantErr    bool
	}{
		{
			name:       "Valid config",
			envVar:     "TEST_CONFIG",
			envValue:   validConfigPath,
			wantConfig: &validConfig,
			wantErr:    false,
		},
		{
			name:       "Missing env var",
			envVar:     "NONEXISTENT_ENV_VAR",
			envValue:   "",
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name:       "File not found",
			envVar:     "TEST_CONFIG",
			envValue:   "/path/to/nonexistent/file.json",
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name:       "Invalid JSON",
			envVar:     "TEST_CONFIG",
			envValue:   invalidJSONPath,
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name:       "Empty config",
			envVar:     "TEST_CONFIG",
			envValue:   emptyConfigPath,
			wantConfig: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			} else if tt.envVar != "" {
				// Make sure the env var isn't set
				os.Unsetenv(tt.envVar)
			}

			// Clean environment variables that affect the test
			os.Unsetenv(LogLevelEnvVar)
			os.Unsetenv(LogToFileEnvVar)

			gotConfig, err := LoadConfig(tt.envVar)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotConfig, tt.wantConfig) {
				t.Errorf("LoadConfig() = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

const (
	// DefaultEnvVar is the environment variable that contains the path to the config file
	DefaultEnvVar = "MCP_CONFIG"
	// LogLevelEnvVar is the environment variable that controls logging level
	LogLevelEnvVar = "MCP_LOG_LEVEL"
	// LogToFileEnvVar is the environment variable that specifies log file path
	LogToFileEnvVar = "MCP_LOG_FILE"
)

// LogLevel represents the log verbosity level
type LogLevel int

const (
	// LogLevelError only logs errors
	LogLevelError LogLevel = iota
	// LogLevelInfo logs info and errors
	LogLevelInfo
	// LogLevelDebug logs everything including debug information
	LogLevelDebug
	// LogLevelTrace logs everything with maximum verbosity
	LogLevelTrace
)

// ToolsConfig represents the tool filtering configuration for a server
type ToolsConfig struct {
	Allowed []string `json:"allowed,omitempty"`
}

// ServerConfig represents the configuration for a single MCP server
type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Tools   *ToolsConfig      `json:"tools,omitempty"` // Optional tool filtering
}

// Config represents the complete configuration for the MCP aggregator
type Config struct {
	Servers  []ServerConfig `json:"servers"`
	LogLevel LogLevel       `json:"-"`
	LogFile  string         `json:"-"`
}

// rawConfig is used to parse different config formats
type rawConfig struct {
	// Array format
	Servers []ServerConfig `json:"servers"`
	// Object format
	MCPServers map[string]struct {
		Command string            `json:"command"`
		Args    []string          `json:"args,omitempty"`
		Env     map[string]string `json:"env,omitempty"`
		Tools   *ToolsConfig      `json:"tools,omitempty"`
	} `json:"mcpServers"`
}

// GetLogLevel returns the configured log level from environment variables
func GetLogLevel() LogLevel {
	levelStr := os.Getenv(LogLevelEnvVar)
	if levelStr == "" {
		return LogLevelInfo // Default to info
	}

	levelInt, err := strconv.Atoi(levelStr)
	if err != nil {
		// Handle string values
		switch levelStr {
		case "error":
			return LogLevelError
		case "info":
			return LogLevelInfo
		case "debug":
			return LogLevelDebug
		case "trace":
			return LogLevelTrace
		default:
			return LogLevelInfo
		}
	}

	// Handle numeric values
	level := LogLevel(levelInt)
	if level < LogLevelError || level > LogLevelTrace {
		return LogLevelInfo
	}
	return level
}

// GetLogFile returns the log file path from environment variables
func GetLogFile() string {
	return os.Getenv(LogToFileEnvVar)
}

// LoadConfig loads the configuration from the specified environment variable
func LoadConfig(envVar string) (*Config, error) {
	if envVar == "" {
		envVar = DefaultEnvVar
	}

	configPath := os.Getenv(envVar)
	if configPath == "" {
		return nil, fmt.Errorf("environment variable %s not set", envVar)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Try to parse the config in different formats
	var raw rawConfig
	if err := json.Unmarshal(configData, &raw); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	var config Config
	config.LogLevel = GetLogLevel()
	config.LogFile = GetLogFile()

	// Check if we have servers in the array format
	if len(raw.Servers) > 0 {
		config.Servers = raw.Servers
	} else if len(raw.MCPServers) > 0 {
		// Convert the object format to our standard format
		for name, server := range raw.MCPServers {
			config.Servers = append(config.Servers, ServerConfig{
				Name:    name,
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
				Tools:   server.Tools,
			})
		}
	}

	if len(config.Servers) == 0 {
		return nil, fmt.Errorf("no servers defined in config")
	}

	// Validate server configuration
	for i, server := range config.Servers {
		if server.Name == "" {
			return nil, fmt.Errorf("server at index %d missing name", i)
		}
		if server.Command == "" {
			return nil, fmt.Errorf("server %s missing command", server.Name)
		}
	}

	return &config, nil
}

// src/config.rs
use serde::Deserialize;
use std::collections::HashMap;
use std::env;
use std::fs;
use std::path::Path;
use anyhow::{Context, Result};

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    #[serde(rename = "mcpServers")]
    pub servers: HashMap<String, ServerConfig>,
    #[serde(rename = "logLevel", default = "default_log_level")]
    pub log_level: String, // We'll parse this into LogLevel enum later
    #[serde(rename = "logFile")]
    pub log_file: Option<String>,
}

impl Config {
    // Apply environment variable overrides to the configuration
    pub fn apply_env_overrides(&mut self) {
        // Override log level from MCP_LOG_LEVEL environment variable
        if let Ok(log_level) = env::var("MCP_LOG_LEVEL") {
            self.log_level = log_level;
        }

        // Override log file from MCP_LOG_FILE environment variable
        if let Ok(log_file) = env::var("MCP_LOG_FILE") {
            self.log_file = Some(log_file);
        }

        // Override server configuration from environment variables
        for (server_name, server_config) in &mut self.servers {
            // Override command from MCP_SERVER_{SERVER_NAME}_COMMAND
            let command_env = format!("MCP_SERVER_{}_COMMAND", server_name.to_uppercase());
            if let Ok(command) = env::var(&command_env) {
                server_config.command = command;
            }

            // Create a collection of env keys to process
            let env_keys: Vec<String> = server_config.env.keys().cloned().collect();
            
            // Override environment variables in the server config
            for env_key in env_keys {
                // Format: MCP_SERVER_{SERVER_NAME}_ENV_{ENV_KEY}
                let env_var_name = format!("MCP_SERVER_{}_ENV_{}", 
                    server_name.to_uppercase(), 
                    env_key.to_uppercase());
                
                if let Ok(env_value) = env::var(&env_var_name) {
                    server_config.env.insert(env_key, env_value);
                }
            }
        }
    }
}

#[derive(Debug, Deserialize, Clone)]
pub struct ServerConfig {
    pub command: String,
    #[serde(default)]
    pub args: Vec<String>,
    #[serde(default)]
    pub env: HashMap<String, String>,
}

fn default_log_level() -> String {
    "info".to_string()
}

// Function to load configuration from a file
pub fn load_config(file_path: &str) -> Result<Config> {
    let path = Path::new(file_path);
    let config_content = fs::read_to_string(path)
        .with_context(|| format!("Failed to read config file from {}", file_path))?;
    let mut config: Config = serde_json::from_str(&config_content)
        .with_context(|| format!("Failed to parse config file from {}", file_path))?;
    
    // Apply environment variable overrides
    config.apply_env_overrides();
    
    Ok(config)
}

// Add tests
#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_config_deserialization() {
        let json_str = r#"
        {
            "mcpServers": {
                "github": {
                    "command": "npx",
                    "args": ["-y", "@modelcontextprotocol/server-github"],
                    "env": {
                        "GITHUB_TOKEN": "test-token"
                    }
                },
                "shortcut": {
                    "command": "npx",
                    "args": ["-y", "@shortcut/mcp"],
                    "env": {
                        "SHORTCUT_API_TOKEN": "test-token"
                    }
                }
            },
            "logLevel": "debug",
            "logFile": "/tmp/mcp.log"
        }
        "#;

        let config: Config = serde_json::from_str(json_str).unwrap();
        
        // Check if the config was deserialized correctly
        assert_eq!(config.log_level, "debug");
        assert_eq!(config.log_file, Some("/tmp/mcp.log".to_string()));
        
        // Check if server configs are present
        assert!(config.servers.contains_key("github"));
        assert!(config.servers.contains_key("shortcut"));
        
        // Check github server config
        let github = &config.servers["github"];
        assert_eq!(github.command, "npx");
        assert_eq!(github.args, vec!["-y", "@modelcontextprotocol/server-github"]);
        assert_eq!(github.env.get("GITHUB_TOKEN"), Some(&"test-token".to_string()));
        
        // Check shortcut server config
        let shortcut = &config.servers["shortcut"];
        assert_eq!(shortcut.command, "npx");
        assert_eq!(shortcut.args, vec!["-y", "@shortcut/mcp"]);
        assert_eq!(shortcut.env.get("SHORTCUT_API_TOKEN"), Some(&"test-token".to_string()));
    }

    #[test]
    fn test_default_log_level() {
        let json_str = r#"
        {
            "mcpServers": {}
        }
        "#;

        let config: Config = serde_json::from_str(json_str).unwrap();
        assert_eq!(config.log_level, "info"); // Should use the default
        assert_eq!(config.log_file, None);
    }

    #[test]
    fn test_load_config_success() -> Result<()> {
        let json_content = r#"
        {
            "mcpServers": {
                "test_server": {
                    "command": "echo",
                    "args": ["hello"],
                    "env": {}
                }
            },
            "logLevel": "info",
            "logFile": "/var/log/mcp.log"
        }
        "#;
        
        let mut temp_file = NamedTempFile::new()?;
        temp_file.write_all(json_content.as_bytes())?;
        let path = temp_file.path().to_str().unwrap();

        let config = load_config(path)?;

        assert_eq!(config.log_level, "info");
        assert_eq!(config.log_file, Some("/var/log/mcp.log".to_string()));
        assert!(config.servers.contains_key("test_server"));
        let server_config = &config.servers["test_server"];
        assert_eq!(server_config.command, "echo");
        assert_eq!(server_config.args, vec!["hello"]);

        Ok(())
    }

    #[test]
    fn test_load_config_file_not_found() {
        let result = load_config("non_existent_file.json");
        assert!(result.is_err());
        // Optionally check the specific error message/type
        assert!(result.unwrap_err().to_string().contains("Failed to read config file"));
    }

    #[test]
    fn test_load_config_invalid_json() -> Result<()> {
        let invalid_json_content = r#"{"mcpServers": {}, "logLevel": "info",,}"#; // Extra comma
        
        let mut temp_file = NamedTempFile::new()?;
        temp_file.write_all(invalid_json_content.as_bytes())?;
        let path = temp_file.path().to_str().unwrap();

        let result = load_config(path);
        assert!(result.is_err());
        // Optionally check the specific error message/type
        assert!(result.unwrap_err().to_string().contains("Failed to parse config file"));

        Ok(())
    }

    #[test]
    fn test_env_var_overrides() -> Result<()> {
        // Setup temporary environment variables
        env::set_var("MCP_LOG_LEVEL", "trace");
        env::set_var("MCP_LOG_FILE", "/env/var/path.log");
        env::set_var("MCP_SERVER_GITHUB_COMMAND", "/usr/bin/custom-github");
        env::set_var("MCP_SERVER_GITHUB_ENV_GITHUB_TOKEN", "env-token-value");
        
        let json_content = r#"
        {
            "mcpServers": {
                "github": {
                    "command": "npx",
                    "args": ["-y", "@modelcontextprotocol/server-github"],
                    "env": {
                        "GITHUB_TOKEN": "file-token"
                    }
                }
            },
            "logLevel": "info",
            "logFile": "/var/log/mcp.log"
        }
        "#;
        
        // Create a temporary file with the JSON content
        let mut temp_file = NamedTempFile::new()?;
        temp_file.write_all(json_content.as_bytes())?;
        let path = temp_file.path().to_str().unwrap();

        // Load the config, which should include environment variable overrides
        let config = load_config(path)?;

        // Check that environment variables took precedence
        assert_eq!(config.log_level, "trace"); // From env var
        assert_eq!(config.log_file, Some("/env/var/path.log".to_string())); // From env var
        
        // Check the github server config
        let github = &config.servers["github"];
        assert_eq!(github.command, "/usr/bin/custom-github"); // From env var
        assert_eq!(github.args, vec!["-y", "@modelcontextprotocol/server-github"]); // From file
        assert_eq!(github.env.get("GITHUB_TOKEN"), Some(&"env-token-value".to_string())); // From env var
        
        // Clean up the environment variables
        env::remove_var("MCP_LOG_LEVEL");
        env::remove_var("MCP_LOG_FILE");
        env::remove_var("MCP_SERVER_GITHUB_COMMAND");
        env::remove_var("MCP_SERVER_GITHUB_ENV_GITHUB_TOKEN");
        
        Ok(())
    }
}

// TODO: Add tests for config loading 
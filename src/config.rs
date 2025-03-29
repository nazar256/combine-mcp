// src/config.rs
use serde::Deserialize;
use std::collections::HashMap;
use anyhow::Result;

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    #[serde(rename = "mcpServers")]
    pub servers: HashMap<String, ServerConfig>,
    #[serde(rename = "logLevel", default = "default_log_level")]
    pub log_level: String, // We'll parse this into LogLevel enum later
    #[serde(rename = "logFile")]
    pub log_file: Option<String>,
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

// TODO: Implement load_config function that reads from file and env vars

// Add tests
#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

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
}

// TODO: Add tests for config loading 
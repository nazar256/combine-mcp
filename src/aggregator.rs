// src/aggregator.rs

use crate::config::Config;
use anyhow::{anyhow, Result};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::collections::HashMap;
use std::process::{Child, Command, Stdio};
use std::sync::Arc;
use tokio::sync::Mutex;
use tracing::{debug, error, info};
use std::path::PathBuf;

// Tool structs matching the actual schemas needed for MCP
#[derive(Debug, Clone, Serialize)]
pub struct Tool {
    pub name: String,
    pub description: String,
    #[serde(rename = "schema")]
    pub input_schema: Value,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CallToolRequest {
    pub params: CallToolParams,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CallToolParams {
    pub name: String,
    pub arguments: Option<Value>,
}

#[derive(Debug, Clone, Serialize)]
pub struct CallToolResult {
    pub content: Vec<ToolResponseContent>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_error: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub meta: Option<Value>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type")]
pub enum ToolResponseContent {
    Text { text: String },
    Json { json: serde_json::Value },
}

// Define a tool mapping struct similar to Go implementation
#[derive(Debug)]
struct ToolMapping {
    name: String,
    path: PathBuf,
    child: Option<Child>,
}

// Define a struct to manage a child process
#[derive(Debug)]
struct MCPClient {
    // For now, just store the child process handle
    // In a real implementation, we would use a proper JSON-RPC client
    child: Child,
    // Additional fields to manage the client would be added here
}

// The main Aggregator struct
pub struct MCPAggregator {
    config: Config,
    clients: Arc<Mutex<HashMap<String, MCPClient>>>,
    tools: Arc<Mutex<HashMap<String, ToolMapping>>>,
}

impl MCPAggregator {
    pub fn new(config: Config) -> Self {
        MCPAggregator {
            config,
            clients: Arc::new(Mutex::new(HashMap::new())),
            tools: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    // Initialize connections to all configured MCP servers
    pub async fn initialize(&self) -> Result<()> {
        info!("Initializing MCP aggregator");
        let mut clients = self.clients.lock().await;
        
        // For each server in config, start the client
        for (server_name, server_config) in &self.config.servers {
            info!("Initializing server: {}", server_name);
            
            // Create the command
            let mut command = Command::new(&server_config.command);
            
            // Add arguments if any
            command.args(&server_config.args);
            
            // Set up pipes for stdin/stdout/stderr
            command
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(Stdio::piped());
            
            // Add environment variables
            for (key, value) in &server_config.env {
                command.env(key, value);
            }
            
            // Start the child process
            debug!("Starting child process: {} {:?}", server_config.command, server_config.args);
            match command.spawn() {
                Ok(child) => {
                    // Store the client
                    clients.insert(server_name.clone(), MCPClient { child });
                    debug!("Child process started successfully for {}", server_name);
                    
                    // TODO: Send initialize request to child and wait for response
                    // TODO: Discover tools from child
                    
                    // For now, just add a dummy tool for demonstration
                    let mut tools = self.tools.lock().await;
                    let tool_name = format!("{}_example_tool", server_name);
                    tools.insert(tool_name.clone(), ToolMapping {
                        name: server_name.clone(),
                        path: PathBuf::new(),
                        child: None,
                    });
                    info!("Added dummy tool: {}", tool_name);
                },
                Err(e) => {
                    error!("Failed to start child process for {}: {}", server_name, e);
                    // Continue with other servers
                }
            }
        }
        
        // Check if we initialized at least one server
        if clients.is_empty() {
            return Err(anyhow!("No servers were successfully initialized"));
        }
        
        info!("MCP aggregator initialized successfully");
        Ok(())
    }

    // Get a list of all available tools from all servers
    pub async fn get_tools(&self) -> Result<Vec<Tool>> {
        let tools_map = self.tools.lock().await;
        
        // For now, just return a list of placeholder tools
        let mut tools = Vec::new();
        
        for (prefixed_name, _mapping) in tools_map.iter() {
            tools.push(Tool {
                name: prefixed_name.clone(),
                description: format!("[{}] Example tool", prefixed_name),
                input_schema: json!({
                    "type": "object",
                    "properties": {
                        "name": {
                            "type": "string",
                            "description": "Example parameter"
                        }
                    },
                    "required": ["name"]
                }),
            });
        }
        
        // Always include sanitize_tool_name tool
        tools.push(Tool {
            name: "sanitize_tool_name".to_string(),
            description: "Sanitizes a tool name by replacing dashes with underscores".to_string(),
            input_schema: json!({
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string",
                        "description": "The tool name to sanitize"
                    }
                },
                "required": ["name"]
            }),
        });
        
        Ok(tools)
    }

    // Call a specific tool by name
    pub async fn call_tool(&self, request: &CallToolRequest) -> Result<CallToolResult> {
        let tool_name = &request.params.name;
        
        // Special case for sanitize_tool_name which is handled directly
        if tool_name == "sanitize_tool_name" {
            let arguments = request.params.arguments.clone().unwrap_or(json!({}));
            if let Some(name) = arguments.get("name").and_then(|n| n.as_str()) {
                let sanitized = sanitize_tool_name(name);
                return Ok(CallToolResult {
                    content: vec![ToolResponseContent::Text {
                        text: sanitized,
                    }],
                    is_error: None,
                    meta: None,
                });
            } else {
                return Err(anyhow!("Missing required 'name' parameter for sanitize_tool_name"));
            }
        }
        
        // For other tools, look up the server and route the request
        let tools_map = self.tools.lock().await;
        match tools_map.get(tool_name) {
            Some(mapping) => {
                debug!("Routing tool call to server: {}", mapping.name);
                
                // TODO: Implement actual routing to the child process
                // For now, just return a placeholder result
                Ok(CallToolResult {
                    content: vec![ToolResponseContent::Text {
                        text: format!("Called {} on server {}", mapping.name, mapping.name),
                    }],
                    is_error: None,
                    meta: None,
                })
            },
            None => Err(anyhow!("Tool not found: {}", tool_name)),
        }
    }

    // Close all child processes
    pub async fn close(&self) -> Result<()> {
        info!("Closing MCP aggregator");
        let mut clients = self.clients.lock().await;
        
        for (name, client) in clients.iter_mut() {
            info!("Shutting down client: {}", name);
            
            // Try to terminate the child process gracefully
            if let Err(e) = client.child.kill() {
                // Ignore errors when process is already dead
                if e.kind() != std::io::ErrorKind::InvalidInput {
                    error!("Error terminating child process for {}: {}", name, e);
                }
            }
        }
        
        // Clear the clients map
        clients.clear();
        
        // Clear the tools map
        let mut tools = self.tools.lock().await;
        tools.clear();
        
        info!("MCP aggregator closed");
        Ok(())
    }
}

// Helper function for tool name sanitization
pub fn sanitize_tool_name(name: &str) -> String {
    name.replace('-', "_")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanitize_tool_name() {
        assert_eq!(sanitize_tool_name("get-user"), "get_user");
        assert_eq!(sanitize_tool_name("create-issue-v2"), "create_issue_v2");
        assert_eq!(sanitize_tool_name("no_dashes"), "no_dashes");
        assert_eq!(sanitize_tool_name("already_sanitized"), "already_sanitized");
    }

    // Additional tests would be added here
} 
// src/server.rs

use crate::aggregator::MCPAggregator;
use anyhow::Result;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::sync::Arc;
use tokio::sync::mpsc;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::select;
use tracing::{debug, error, info};

// JSON-RPC 2.0 request structure
#[derive(Debug, Deserialize)]
struct JsonRpcRequest {
    method: String,
    id: Option<serde_json::Value>,
    params: Option<serde_json::Value>,
}

// JSON-RPC 2.0 response structure
#[derive(Debug, Serialize)]
struct JsonRpcResponse {
    jsonrpc: String,
    id: Value,
    result: Option<Value>,
    error: Option<JsonRpcError>,
}

// JSON-RPC 2.0 error structure
#[derive(Debug, Serialize)]
struct JsonRpcError {
    code: i32,
    message: String,
    data: Option<Value>,
}

impl JsonRpcResponse {
    fn success(id: Value, result: Value) -> Self {
        JsonRpcResponse {
            jsonrpc: "2.0".to_string(),
            id,
            result: Some(result),
            error: None,
        }
    }

    fn error(id: Value, code: i32, message: String, data: Option<Value>) -> Self {
        JsonRpcResponse {
            jsonrpc: "2.0".to_string(),
            id,
            result: None,
            error: Some(JsonRpcError {
                code,
                message,
                data,
            }),
        }
    }

    fn method_not_found(id: Value) -> Self {
        Self::error(id, -32601, "Method not found".to_string(), None)
    }

    fn internal_error(id: Value, message: String) -> Self {
        Self::error(id, -32603, message, None)
    }
}

pub async fn run(aggregator: Arc<MCPAggregator>) -> Result<()> {
    info!("Starting MCP server over stdio");
    
    // Use stdin for reading
    let stdin = tokio::io::stdin();
    let mut reader = BufReader::new(stdin);
    
    // Use stdout for writing
    let mut stdout = tokio::io::stdout();
    
    // Create a shutdown channel
    let (shutdown_tx, mut shutdown_rx) = mpsc::channel::<()>(1);
    
    // Process requests line by line
    let mut buffer = String::new();
    loop {
        buffer.clear();
        
        // Wait for either a line to be read or a shutdown signal
        select! {
            result = reader.read_line(&mut buffer) => {
                match result {
                    Ok(0) => {
                        // EOF, exit the loop
                        info!("End of input, shutting down");
                        break;
                    }
                    Ok(_) => {
                        // Process the request
                        let request_str = buffer.trim();
                        debug!("Received request: {}", request_str);
                        
                        // Parse and process the request
                        let response = match serde_json::from_str::<JsonRpcRequest>(request_str) {
                            Ok(request) => {
                                debug!("Parsed request: {:?}", request);
                                
                                // Check if this is a shutdown request
                                if request.method == "$/shutdown" {
                                    info!("Received shutdown request");
                                    // Send shutdown signal
                                    let _ = shutdown_tx.send(()).await;
                                }
                                
                                process_request(&request, &aggregator).await
                            }
                            Err(err) => {
                                error!("Failed to parse request: {}", err);
                                JsonRpcResponse::error(
                                    json!(null),
                                    -32700,
                                    format!("Parse error: {}", err),
                                    None,
                                )
                            }
                        };
                        
                        // Send the response
                        let response_str = serde_json::to_string(&response)?;
                        debug!("Sending response: {}", response_str);
                        stdout.write_all(response_str.as_bytes()).await?;
                        stdout.write_all(b"\n").await?;
                        stdout.flush().await?;
                    }
                    Err(e) => {
                        error!("Error reading from stdin: {}", e);
                        break;
                    }
                }
            }
            _ = shutdown_rx.recv() => {
                info!("Shutdown signal received, stopping server");
                break;
            }
        }
    }
    
    info!("MCP server finished");
    Ok(())
}

async fn process_request(request: &JsonRpcRequest, aggregator: &Arc<MCPAggregator>) -> JsonRpcResponse {
    // Default ID to use for responses
    let id = request.id.clone().unwrap_or(json!(null));
    
    // Process based on method
    match request.method.as_str() {
        "$/initialize" => {
            info!("Received initialize request");
            JsonRpcResponse::success(id, json!({
                "serverInfo": {
                    "name": "combine-mcp-rust",
                    "version": "0.1.0"
                }
            }))
        }
        
        "$/shutdown" => {
            info!("Received shutdown request");
            // Initiate graceful shutdown
            JsonRpcResponse::success(id, json!(null))
        }
        
        "$/exit" => {
            info!("Received exit notification");
            // Notifications don't need responses, but for consistency in our code:
            JsonRpcResponse::success(id, json!(null))
        }
        
        "tools/list" => handle_list_tools(id, aggregator).await,
        
        "tools/call" => {
            if let Some(params) = &request.params {
                handle_call_tool(id, params, aggregator).await
            } else {
                JsonRpcResponse::error(
                    id,
                    -32602,
                    "Invalid params: params are required for tools/call".to_string(),
                    None,
                )
            }
        }
        
        _ => {
            error!("Method not found: {}", request.method);
            JsonRpcResponse::method_not_found(id)
        }
    }
}

async fn handle_list_tools(id: Value, aggregator: &Arc<MCPAggregator>) -> JsonRpcResponse {
    // Get tools from the aggregator
    match aggregator.get_tools().await {
        Ok(tools) => {
            // Convert tools to the expected JSON format
            JsonRpcResponse::success(id, json!({ "tools": tools }))
        },
        Err(e) => {
            error!("Error getting tools: {}", e);
            JsonRpcResponse::internal_error(id, format!("Failed to get tools: {}", e))
        }
    }
}

async fn handle_call_tool(id: Value, params: &Value, aggregator: &Arc<MCPAggregator>) -> JsonRpcResponse {
    // Extract the tool name and arguments
    let tool_name = match params.get("name") {
        Some(name) => name.as_str(),
        None => return JsonRpcResponse::error(
            id,
            -32602,
            "Invalid params: missing 'name' field".to_string(),
            None,
        ),
    };
    
    let arguments = params.get("arguments").cloned().unwrap_or(json!({}));
    
    // Create the CallToolRequest
    let request = crate::aggregator::CallToolRequest {
        params: crate::aggregator::CallToolParams {
            name: tool_name.unwrap_or_default().to_string(),
            arguments: Some(arguments),
        },
    };
    
    // Call the tool via the aggregator
    match aggregator.call_tool(&request).await {
        Ok(result) => {
            JsonRpcResponse::success(id, json!(result))
        },
        Err(e) => {
            error!("Error calling tool {}: {}", tool_name.unwrap_or("unknown"), e);
            JsonRpcResponse::error(
                id,
                -32603,
                format!("Tool call failed: {}", e),
                None,
            )
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_json_rpc_response_success() {
        let id = json!(1);
        let result = json!({"value": "test"});
        let response = JsonRpcResponse::success(id.clone(), result.clone());
        
        assert_eq!(response.jsonrpc, "2.0");
        assert_eq!(response.id, id);
        assert_eq!(response.result, Some(result));
        assert!(response.error.is_none());
    }

    #[test]
    fn test_json_rpc_response_error() {
        let id = json!(1);
        let code = -32600;
        let message = "Invalid Request".to_string();
        let response = JsonRpcResponse::error(id.clone(), code, message.clone(), None);
        
        assert_eq!(response.jsonrpc, "2.0");
        assert_eq!(response.id, id);
        assert!(response.result.is_none());
        assert!(response.error.is_some());
        
        let error = response.error.unwrap();
        assert_eq!(error.code, code);
        assert_eq!(error.message, message);
        assert!(error.data.is_none());
    }

    #[test]
    fn test_json_rpc_method_not_found() {
        let id = json!(1);
        let response = JsonRpcResponse::method_not_found(id.clone());
        
        assert_eq!(response.jsonrpc, "2.0");
        assert_eq!(response.id, id);
        assert!(response.result.is_none());
        assert!(response.error.is_some());
        
        let error = response.error.unwrap();
        assert_eq!(error.code, -32601);
        assert_eq!(error.message, "Method not found");
        assert!(error.data.is_none());
    }

    #[test]
    fn test_json_rpc_internal_error() {
        let id = json!(1);
        let message = "Server error".to_string();
        let response = JsonRpcResponse::internal_error(id.clone(), message.clone());
        
        assert_eq!(response.jsonrpc, "2.0");
        assert_eq!(response.id, id);
        assert!(response.result.is_none());
        assert!(response.error.is_some());
        
        let error = response.error.unwrap();
        assert_eq!(error.code, -32603);
        assert_eq!(error.message, message);
        assert!(error.data.is_none());
    }
} 
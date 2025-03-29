// src/error.rs
use thiserror::Error;

#[derive(Error, Debug)]
pub enum AppError {
    #[error("Configuration Error: {0}")]
    Config(String),

    #[error("IO Error: {0}")]
    Io(#[from] std::io::Error),

    #[error("MCP Protocol Error: {0}")]
    McpProtocol(String),

    #[error("Child Process Error: {0}")]
    ChildProcess(String),

    #[error("Logging Setup Error: {0}")]
    Logging(String),

    #[error("Initialization Error: {0}")]
    Initialization(String),

    #[error("Tool Not Found: {0}")]
    ToolNotFound(String),

    #[error("JSON Parse Error: {0}")]
    Json(#[from] serde_json::Error),

    #[error(transparent)]
    Other(#[from] anyhow::Error), // Generic error wrapper
} 
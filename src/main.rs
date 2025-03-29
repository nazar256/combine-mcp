use anyhow::Result;
use std::sync::Arc;
use std::collections::HashMap;
use tokio::signal;
use tracing::{error, info};

// Import our own modules
mod config;
mod logger;
mod aggregator;
mod server;
mod error;

use crate::aggregator::MCPAggregator;
use crate::config::load_config;

#[tokio::main]
async fn main() -> Result<()> {
    dotenvy::dotenv().ok(); // Load .env file if present

    // Load configuration
    let config_path = std::env::var("MCP_CONFIG_PATH").unwrap_or_else(|_| "config.json".to_string());
    let cfg = match load_config(&config_path) {
        Ok(config) => config,
        Err(err) => {
            eprintln!("Error loading config from {}: {}", config_path, err);
            // Fallback to a minimal config
            config::Config {
                servers: HashMap::new(),
                log_level: "info".to_string(),
                log_file: None,
            }
        }
    };

    // Setup logging
    logger::setup_logger(&cfg)?;
    info!("Combine MCP (Rust) Starting...");

    // Create and initialize the aggregator
    let aggregator = Arc::new(MCPAggregator::new(cfg));
    
    // Initialize the aggregator
    // Note: We can continue even if initialization fails, as we have the sanitize_tool_name built-in
    if let Err(err) = aggregator.initialize().await {
        error!("Failed to initialize aggregator: {}", err);
        info!("Continuing with limited functionality");
    }

    // Clone for the server task
    let aggregator_clone = aggregator.clone();
    
    // Start the server in a separate task
    let server_handle = tokio::spawn(async move {
        if let Err(e) = server::run(aggregator_clone).await {
            error!("Server error: {}", e);
        }
    });

    // Wait for Ctrl+C signal
    if let Err(e) = signal::ctrl_c().await {
        error!("Failed to listen for ctrl+c: {}", e);
    } else {
        info!("Received ctrl+c, initiating shutdown...");
    }
    
    // Clean up the aggregator
    if let Err(err) = aggregator.close().await {
        error!("Error closing aggregator: {}", err);
    }

    // Wait for the server to finish (it should detect stdin is closed)
    if let Err(e) = server_handle.await {
        error!("Error joining server task: {}", e);
    }

    info!("Combine MCP (Rust) Shutting down.");
    Ok(())
} 
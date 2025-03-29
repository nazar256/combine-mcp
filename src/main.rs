use anyhow::Result;
use std::sync::Arc;
use std::collections::HashMap;

// Import our own modules
mod config;
mod logger;
mod aggregator;
mod server;
mod error;

use crate::aggregator::MCPAggregator;

#[tokio::main]
async fn main() -> Result<()> {
    dotenvy::dotenv().ok(); // Load .env file if present

    // TODO: Proper config loading (using config crate)
    let cfg = config::Config {
        servers: HashMap::new(), // Placeholder
        log_level: "info".to_string(),
        log_file: None,
    };

    // Setup logging
    logger::init(&cfg)?;
    println!("Combine MCP (Rust) Starting..."); // Using println instead of tracing for now

    // Create and initialize the aggregator
    let aggregator = Arc::new(MCPAggregator::new(cfg));
    // TODO: Call aggregator.initialize() - needs context/async handling
    // aggregator.initialize().await?;

    // Start the MCP server
    // TODO: Handle server result and shutdown
    // server::run(aggregator.clone()).await?;

    // TODO: Add signal handling for graceful shutdown
    // aggregator.close().await?;

    println!("Combine MCP (Rust) Shutting down.");
    Ok(())
} 
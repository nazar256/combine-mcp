// src/logger.rs

use crate::config::Config;
use anyhow::{Context, Result};
use std::str::FromStr;
use tracing::{Level, info, debug, trace};
use tracing_subscriber::{
    fmt, 
    prelude::*, 
    EnvFilter, 
    filter::LevelFilter
};
use tracing_appender::rolling::{RollingFileAppender, Rotation};

/// Initializes the logger based on the provided configuration.
/// - Sets up console logging with appropriate formatting
/// - Sets up file logging if a log file is specified in the config
/// - Configures the log level based on the config
pub fn setup_logger(config: &Config) -> Result<()> {
    // Parse the log level from the config
    let level = parse_log_level(&config.log_level)
        .with_context(|| format!("Invalid log level: {}", config.log_level))?;
    
    // Create a filter that includes logs at the specified level and above
    let filter = EnvFilter::from_default_env()
        .add_directive(LevelFilter::from_level(level).into());
    
    // Configure stderr logging (changed from stdout to avoid interfering with JSON-RPC)
    let stderr_layer = fmt::layer()
        .with_writer(std::io::stderr)
        .with_thread_ids(true)
        .with_target(true)
        .compact();
    
    // If a log file is specified, set up file logging
    if let Some(log_file) = &config.log_file {
        // Set up rolling file logger (daily rotation)
        let file_appender = RollingFileAppender::new(
            Rotation::DAILY, 
            std::path::Path::new(log_file).parent().unwrap_or_else(|| std::path::Path::new(".")), 
            std::path::Path::new(log_file).file_name().unwrap_or_default(),
        );
        
        let file_layer = fmt::layer()
            .with_writer(file_appender)
            .with_ansi(false) // No ANSI colors in log files
            .with_thread_ids(true) 
            .with_target(true);
        
        // Register both console and file subscribers
        tracing_subscriber::registry()
            .with(filter)
            .with(stderr_layer)
            .with(file_layer)
            .try_init()
            .with_context(|| "Failed to initialize tracing subscriber")?;
        
        info!("Logging initialized at level {} with output to stderr and file: {}", level, log_file);
    } else {
        // Register console subscriber only
        tracing_subscriber::registry()
            .with(filter)
            .with(stderr_layer)
            .try_init()
            .with_context(|| "Failed to initialize tracing subscriber")?;
        
        info!("Logging initialized at level {} with output to stderr only", level);
    }
    
    debug!("Debug logging enabled");
    trace!("Trace logging enabled");
    
    Ok(())
}

/// Parses a string log level into a tracing::Level.
fn parse_log_level(level_str: &str) -> Result<Level> {
    match level_str.to_lowercase().as_str() {
        "error" => Ok(Level::ERROR),
        "warn" => Ok(Level::WARN),
        "info" => Ok(Level::INFO),
        "debug" => Ok(Level::DEBUG),
        "trace" => Ok(Level::TRACE),
        _ => Level::from_str(level_str)
            .with_context(|| format!("Invalid log level: {}", level_str)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_parse_log_level() {
        assert_eq!(parse_log_level("error").unwrap(), Level::ERROR);
        assert_eq!(parse_log_level("warn").unwrap(), Level::WARN);
        assert_eq!(parse_log_level("info").unwrap(), Level::INFO);
        assert_eq!(parse_log_level("debug").unwrap(), Level::DEBUG);
        assert_eq!(parse_log_level("trace").unwrap(), Level::TRACE);
        
        // Case insensitive
        assert_eq!(parse_log_level("ERROR").unwrap(), Level::ERROR);
        assert_eq!(parse_log_level("Debug").unwrap(), Level::DEBUG);
        
        // Invalid level
        assert!(parse_log_level("invalid").is_err());
    }
    
    // Note: Testing the actual setup_logger function is challenging
    // because tracing_subscriber::try_init can only be called once per process.
    // In a real-world scenario, we'd use integration tests or create
    // a mock/test version of the tracing subscriber.
} 
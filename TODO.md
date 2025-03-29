# Combine MCP Rust Implementation TODO List

This file tracks the remaining tasks to complete the Rust implementation of `combine-mcp`.

## Core Implementation

- [x] **Configuration (`config.rs`)**
    - [x] Implement `load_config` function to read configuration from a specified file (e.g., `config.json`).
    - [x] Add support for overriding configuration values with environment variables.
    - [x] Add tests for file loading scenarios (file not found, invalid format).
    - [x] Add tests for environment variable overrides.
- [x] **Logging (`logger.rs`)**
    - [x] Implement `setup_logger` using the `tracing` and `tracing-subscriber` crates.
    - [x] Configure logging level (e.g., debug, info, warn, error) based on the loaded configuration.
    - [x] Configure logging output (e.g., stdout, file) based on the loaded configuration.
    - [x] Add tests for logger configuration.
    - [x] Use stderr for logging to avoid interfering with JSON-RPC over stdout.
- [ ] **Aggregator (`aggregator.rs`)**
    - [x] Implement `Aggregator::new` to initialize the aggregator state.
    - [x] Implement `Aggregator::start_tool` to spawn and manage child tool processes.
        - [x] Handle executable path resolution.
        - [x] Capture and manage child process stdin/stdout/stderr.
    - [x] Implement `Aggregator::stop_tool` to terminate a specific child process.
    - [x] Implement `Aggregator::route_message` to forward messages between the main server and the correct child tool process based on the tool name.
    - [x] Implement `Aggregator::handle_child_exit` to manage cleanup when a child process terminates unexpectedly.
    - [x] Implement `Aggregator::find_child_by_pid` or similar logic for internal management.
    - [ ] Add comprehensive tests for aggregator logic (starting, stopping, routing, error handling).
- [x] **Server (`server.rs`)**
    - [x] Implement `start_server` to run the main JSON-RPC server loop over stdio.
    - [x] Implement JSON-RPC message parsing (Requests, Notifications) using `serde_json`.
    - [x] Implement `handle_message` logic:
        - [x] Route `$/initialize` requests.
        - [x] Route `$/shutdown` requests.
        - [x] Route `$/exit` notifications.
        - [x] Route tool-specific requests/notifications via placeholder implementation.
    - [x] Handle sending JSON-RPC responses and notifications back to the client (Cursor).
    - [x] Implement graceful shutdown logic.
    - [x] Add tests for server message handling and response formatting.
- [x] **Main (`main.rs`)**
    - [x] Integrate `load_config` (Added setup but using placeholder until config implemented).
    - [x] Integrate `setup_logger` calls.
    - [x] Integrate `start_server` calls.
    - [x] Implement top-level error handling and application exit logic.
    - [x] Implement signal handling for graceful shutdown (e.g., SIGINT, SIGTERM).

## JSON-RPC Implementation (MCP Protocol)

Tests with `mcptools` revealed that our implementation was missing the JSON-RPC functionality required by the MCP protocol:

- [x] **JSON-RPC Message Handling**
    - [x] Implement JSON-RPC 2.0 request parsing and response formatting.
    - [x] Add support for required MCP methods (`$/initialize`, `$/shutdown`, `$/exit`, `tools/list`, `tools/call`).
    - [x] Add error handling for invalid requests and server errors.
- [x] **Tool Registration**
    - [x] Define tool schemas for exposure via the `tools/list` endpoint (currently placeholder).
    - [x] Implement tool metadata (name, description, parameters, etc.) (currently placeholder).
    - [x] Add support for tool execution via the `tools/call` endpoint (currently placeholder).
- [ ] **Resource Support** (optional)
    - [ ] Implement resource listing and retrieval.
    - [ ] Handle resource templates and URI parsing.

## External Testing

- [x] **Testing with mcptools**
    - [x] Fix logger to use stderr to avoid interfering with JSON-RPC over stdout.
    - [x] Successfully test `tools/list` with mcptools to list available tools.
    - [x] Successfully test `tools/call` with mcptools to call a specific tool.

## Integration of Server with Aggregator

- [x] Replace placeholder implementations with actual calls to Aggregator methods:
    - [x] Connect `handle_list_tools` to `Aggregator::get_tools`
    - [x] Connect `handle_call_tool` to `Aggregator::call_tool`
    - [x] Implement shutdown initiation in `$/shutdown` handler

## Testing and Quality

- [ ] Ensure all implemented functions have corresponding unit or integration tests.
- [ ] Achieve reasonable test coverage.
- [ ] Resolve all compiler warnings (`unused_imports`, `unused_variables`, etc.).
- [ ] Run `cargo clippy` and address lints.
- [ ] Run `cargo fmt` to ensure consistent code style.

## Documentation

- [ ] Update `README.md` with detailed usage instructions for the Rust version if needed.
- [ ] Add comments to explain complex logic within the code.

Let's proceed with implementing these tasks step by step. 
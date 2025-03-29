# MCP Aggregator

An MCP (Model Context Protocol) aggregator that allows you to combine multiple MCP servers into a single interface. 

## Why combine MCP servers into one?

The primary reason for this app was to work around Cursor's limitation of only being able to use 2 MCP servers at a time. When adding a 3rd MCP server, it breaks the ability to use one of the other two.

## Overview

The MCP Aggregator acts as a bridge between Cursor (or any other MCP client) and multiple MCP servers. It functions both as an MCP server (when talking to Cursor) and as an MCP client (when talking to backend MCP servers).

Key features:
- Provides a stdio interface for Cursor and other MCP clients
- Connects to multiple backend MCP servers
- Prefixes methods from backend servers (e.g., "shortcut_search_stories" for "search_stories" method from a "shortcut" MCP)
- Automatically sanitizes tool names by replacing dashes with underscores for Cursor compatibility
- Configurable via environment variables and JSON config file
- Debug logging with configurable levels

## Installation

### Go Implementation (Stable)

#### Prerequisites

- Go 1.21 or higher

#### Using go install (recommended)

```bash
# Install directly from GitHub (binary will be placed in $GOPATH/bin)
go install github.com/nazar256/combine-mcp/cmd/combine-mcp@latest

# Ensure $GOPATH/bin is in your PATH
# For example, add this to your .bashrc or .zshrc:
# export PATH=$PATH:$(go env GOPATH)/bin
```

#### Using the Makefile

The project includes a Makefile for common tasks:

```bash
# Build the binary
make build

# Run tests
make test

# Clean up build artifacts
make clean
```

### Rust Implementation (In Development)

#### Prerequisites

- Rust 1.70 or higher (with Cargo)

#### Building from source

```bash
# Clone the repository
git clone https://github.com/nazar256/combine-mcp.git
cd combine-mcp

# Build the Rust implementation
cargo build --release

# Run the executable
./target/release/combine-mcp-rust
```

#### Running tests

```bash
# Run the tests
cargo test
```

## Usage

### Configure the aggregator

Basically you can copy existing Cursor MCP config to a location of your choice, let's say `~/.config/mcp/config.json`. It should look like this:

```json
{
  "mcpServers": {
    "shortcut": {
      "command": "npx",
      "args": ["-y", "@shortcut/mcp"],
      "env": {
        "SHORTCUT_API_TOKEN": "your-shortcut-api-token-here"
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "your-github-token-here"  
      }
    }
  }
}
```

### Configure the aggregator in Cursor

Now in Cursor config you may leave the only one MCP server - aggregator. The config may look like this (assuming you have `combine-mcp` binary is installed in your PATH and you have `~/.config/mcp/config.json` file):

```json
{
  "mcpServers": {
    "aggregator": {
      "command": "combine-mcp",
      "env": {
        "MCP_CONFIG": "~/.config/mcp/config.json"
      }
    }
  }
}
```

### Environment Variables

- `MCP_CONFIG`: Path to the configuration file (required)
- `MCP_LOG_LEVEL`: Logging level (error, info, debug, trace) - default: info
- `MCP_LOG_FILE`: Path to the log file
- `MCP_PROTOCOL_VERSION`: Force a specific protocol version for compatibility
- `MCP_CURSOR_MODE`: Enable Cursor-specific compatibility adjustments

## Tool Name Sanitization

The MCP Aggregator automatically sanitizes tool names by replacing dashes with underscores. This is necessary because Cursor has a known issue where it cannot properly detect or use tools with dashes in their names.

For example:
- Original tool name: `get-user`
- Sanitized tool name: `get_user`
- Prefixed tool name (for shortcut server): `shortcut_get_user`

The sanitization is transparent - when you call a tool using the sanitized name, the aggregator maps it back to the original name when forwarding the request to the backend server.

### Testing

For testing purposes, use:

```bash
# Go implementation
make test

# Rust implementation
cargo test
```

## License

MIT 

# MCP Aggregator

An MCP (Model Context Protocol) aggregator that allows you to combine multiple MCP servers into a single interface. The code is mostly AI-generated.

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

### Prerequisites

- Go 1.21 or higher
- [mcp-go](https://github.com/mark3labs/mcp-go) package

### Using go install (recommended)

```bash
# Install directly from GitHub (binary will be placed in $GOPATH/bin)
go install github.com/yourusername/combine-mcp/cmd@latest

# Ensure $GOPATH/bin is in your PATH
# For example, add this to your .bashrc or .zshrc:
# export PATH=$PATH:$(go env GOPATH)/bin
```

### Using the Makefile

The project includes a Makefile for common tasks:

```bash
# Build the binary
make build

# Run tests
make test

# Clean up build artifacts
make clean
```

## Usage

When connecting this aggregator (i.e. in Cursor, Windsurf or Claude), configure the aggregator using the `MCP_CONFIG` environment variable, which should point to a JSON configuration file.

Example configuration file (`config.example.json`):
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
make test
```


## License

MIT 
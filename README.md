# MCP Aggregator

An MCP (Model Context Protocol) aggregator that allows you to combine multiple MCP servers into a single interface. The code is mostly AI-generated.

## Why combine MCP servers into one?

The primary reason for this app was to work around Cursor's limitation of only being able to use 2 MCP servers at a time. No matter which, in my case when I added 3rd MCP server, it was breaking the ability to use one of other two.

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

### Using install script (recommended)

You can install the latest version of combine-mcp using our installation script:

```bash
# Download and run the installation script
curl -fsSL https://raw.githubusercontent.com/nazar256/combine-mcp/main/install.sh | bash

# Or install a specific version
curl -fsSL https://raw.githubusercontent.com/nazar256/combine-mcp/main/install.sh | bash -s -- -v v1.0.0
```

The script will:
- Detect your operating system and architecture
- Download the appropriate pre-compiled binary
- Verify the checksum
- Install it to a suitable location in your PATH
- Make it executable

### Using go install (alternative)

```bash
# Install directly from GitHub (binary will be placed in $GOPATH/bin)
go install github.com/nazar256/combine-mcp/cmd/combine-mcp@latest

# Ensure $GOPATH/bin is in your PATH
# For example, add this to your .bashrc or .zshrc:
# export PATH=$PATH:$(go env GOPATH)/bin
```

### Using Docker (alternative)

You can run combine-mcp directly using Docker without installing it locally:

```bash
# Run the latest version
docker run --rm -v ~/.config/mcp:/config ghcr.io/nazar256/combine-mcp:latest

# Run a specific version
docker run --rm -v ~/.config/mcp:/config ghcr.io/nazar256/combine-mcp:v1.0.0

# Set environment variables
docker run --rm -v ~/.config/mcp:/config -e MCP_CONFIG=/config/config.json -e MCP_LOG_LEVEL=debug ghcr.io/nazar256/combine-mcp:latest
```

To use it with Cursor, you'd need to configure the MCP server to use Docker:

```json
{
  "mcpServers": {
    "aggregator": {
      "command": "docker",
      "args": ["run", "--rm", "-v", "~/.config/mcp:/config", "ghcr.io/nazar256/combine-mcp:latest"],
      "env": {
        "MCP_CONFIG": "/config/config.json"
      }
    }
  }
}
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

### Configure the aggregator

Basically you can copy existing Cursor MCP config to a location of your choice, let's say `~/.config/mcp/config.json`. It should look like this:

Nice feature for Cursor users is filtering tools from MCP servers. You can manage tools to not reach the limit of 40 tools in Cursor and not expose the ones you don't want Cursor to use.

```json
{
  "mcpServers": {
    "shortcut": {
      "command": "npx",
      "args": ["-y", "@shortcut/mcp"],
      "env": {
        "SHORTCUT_API_TOKEN": "your-shortcut-api-token-here"
      },
      "tools": {
        "allowed": ["search-stories", "get-story", "create-story"]
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

Now in Cursor config you may leave the only one MCP server - aggregator. The config may look like this (assuming you have `combine-mcp` binary is instlaled your PATH and you have `~/.config/mcp/config.json` file):

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

### Tool Filtering

The MCP Aggregator supports optional tool filtering per server. This is useful when you want to:
- Limit the number of exposed tools to stay within Cursor's tool limit (40 tools maximum)
- Only expose specific tools from each server
- Avoid tool name conflicts between servers
- Improve performance by reducing the number of tools to process

To enable tool filtering, add a `tools` object to your server configuration with an `allowed` array listing the tools you want to expose:

```json
{
  "mcpServers": {
    "shortcut": {
      "command": "npx",
      "args": ["-y", "@shortcut/mcp"],
      "env": {
        "SHORTCUT_API_TOKEN": "your-shortcut-api-token-here"
      },
      "tools": {
        "allowed": [
          "search-stories",
          "get-story",
          "create-story",
          "assign-current-user-as-owner"
        ]
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "your-github-token-here"
      },
      "tools": {
        "allowed": [
          "create-pr",
          "list-prs",
          "get-pr",
          "merge-pr"
        ]
      }
    }
  }
}
```

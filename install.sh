#!/bin/bash
set -e

# Print error and exit
error() {
    echo "Error: $1" >&2
    exit 1
}

# Check for required dependencies
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        error "curl is required but not installed. Please install curl and try again."
    fi
}

# Get latest release info from GitHub API
get_latest_release() {
    curl --silent "https://api.github.com/repos/nazar256/combine-mcp/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

# Detect OS and architecture
detect_platform() {
    local OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    local ARCH="$(uname -m)"
    
    # Convert architecture to Go format
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            error "Unsupported architecture: $ARCH. This script supports x86_64/amd64 and arm64 architectures."
            ;;
    esac
    
    # Convert OS to Go format
    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        *)
            error "Unsupported operating system: $OS. This script supports Linux and macOS."
            ;;
    esac
    
    echo "$OS-$ARCH"
}

# Check if user has sudo access (for system directories)
has_sudo() {
    if command -v sudo > /dev/null; then
        # Try sudo with a simple command
        if sudo -n true 2>/dev/null; then
            return 0 # User has sudo without password
        else
            # Sudo requires password - only prompt if needed
            return 0 # Return success but will only prompt later if actually needed
        fi
    else
        return 1 # sudo command not found
    fi
}

# Find suitable installation directory
find_install_dir() {
    local OS="$1"
    local CAN_USE_SUDO=$2
    local INSTALL_DIRS=()
    
    # Home directory paths (no sudo required)
    if [ -n "$HOME" ]; then
        if [ "$OS" = "darwin" ]; then
            # macOS preferred paths
            INSTALL_DIRS+=("$HOME/.local/bin")
            INSTALL_DIRS+=("$HOME/bin")
        else
            # Linux preferred paths
            INSTALL_DIRS+=("$HOME/.local/bin")
            INSTALL_DIRS+=("$HOME/bin")
        fi
    fi
    
    # Check each user directory first
    for DIR in "${INSTALL_DIRS[@]}"; do
        # Create directory if it doesn't exist
        if [ ! -d "$DIR" ]; then
            mkdir -p "$DIR" || continue
        fi
        
        # Check if directory is writable
        if [ -w "$DIR" ]; then
            echo "$DIR"
            return 0
        fi
    done
    
    # If no home directory is available/writable, try system directories
    if [ "$CAN_USE_SUDO" = "true" ]; then
        # Only now do we prompt for sudo password if needed
        echo "No writable user directories found. Some installation locations require sudo access."
        echo "Please enter your password if prompted."
        if ! sudo -v 2>/dev/null; then
            echo "Cannot use sudo. Falling back to user directory."
        else
            # System paths that require sudo
            local SYSTEM_DIRS=()
            if [ "$OS" = "darwin" ]; then
                # macOS system paths
                SYSTEM_DIRS+=("/usr/local/bin")
            else
                # Linux system paths
                SYSTEM_DIRS+=("/usr/local/bin")
                SYSTEM_DIRS+=("/usr/bin")
            fi
            
            for DIR in "${SYSTEM_DIRS[@]}"; do
                # Create directory if it doesn't exist
                if [ ! -d "$DIR" ]; then
                    sudo mkdir -p "$DIR" || continue
                fi
                
                echo "$DIR"
                return 0
            done
        fi
    fi
    
    # If we got here, create and use ~/.local/bin as fallback
    mkdir -p "$HOME/.local/bin"
    echo "$HOME/.local/bin"
    
    # Show instructions for adding to PATH if the directory might not be in PATH
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        echo "Note: You may need to add $HOME/.local/bin to your PATH."
        echo "Add the following line to your shell configuration file (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
}

# Download binary from GitHub
download_binary() {
    local VERSION=$1
    local PLATFORM=$2
    local OUTPUT_DIR=$3
    local BINARY_NAME="combine-mcp"
    local TEMP_DIR=$(mktemp -d)
    
    echo "Downloading combine-mcp $VERSION for $PLATFORM..."
    
    # Set the file extension based on OS
    local URL="https://github.com/nazar256/combine-mcp/releases/download/$VERSION/${BINARY_NAME}-${PLATFORM}.gz"
    local CHECKSUM_URL="https://github.com/nazar256/combine-mcp/releases/download/$VERSION/${BINARY_NAME}-${PLATFORM}.sha256"
    
    # Download binary and checksum
    curl -L "$URL" -o "$TEMP_DIR/$BINARY_NAME.gz" || error "Failed to download binary"
    curl -L "$CHECKSUM_URL" -o "$TEMP_DIR/$BINARY_NAME.sha256" || error "Failed to download checksum"
    
    # Decompress binary
    echo "Decompressing binary..."
    gzip -d "$TEMP_DIR/$BINARY_NAME.gz" || error "Failed to decompress binary"
    
    # Verify checksum
    cd "$TEMP_DIR"
    if command -v sha256sum > /dev/null; then
        # Extract just the hash from the checksum file (in case it includes filename)
        EXPECTED_HASH=$(cat "$BINARY_NAME.sha256" | awk '{print $1}')
        ACTUAL_HASH=$(sha256sum "$BINARY_NAME" | awk '{print $1}')
        
        if [[ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]]; then
            error "Checksum verification failed"
        fi
    elif command -v shasum > /dev/null; then
        # macOS uses shasum instead of sha256sum
        EXPECTED_HASH=$(cat "$BINARY_NAME.sha256" | awk '{print $1}')
        ACTUAL_HASH=$(shasum -a 256 "$BINARY_NAME" | awk '{print $1}')
        
        if [[ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]]; then
            error "Checksum verification failed"
        fi
    else
        echo "Warning: Cannot verify checksum, neither sha256sum nor shasum found"
    fi
    
    # Make binary executable
    chmod +x "$BINARY_NAME"
    
    # Install binary
    DEST_PATH="$OUTPUT_DIR/$BINARY_NAME"
    if [[ -w "$OUTPUT_DIR" ]]; then
        echo "Installing to $DEST_PATH..."
        mv "$BINARY_NAME" "$DEST_PATH"
    else
        echo "Installing to $DEST_PATH (requires sudo)..."
        sudo mv "$BINARY_NAME" "$DEST_PATH"
    fi
    
    # Cleanup
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
    
    echo "Successfully installed combine-mcp to $DEST_PATH"
}

# Main installation procedure
main() {
    # Check dependencies
    check_dependencies
    
    # Parse command line arguments
    local VERSION="latest"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -h|--help)
                echo "Usage: ./install.sh [options]"
                echo "Options:"
                echo "  -v, --version VERSION  Install specific version (default: latest)"
                echo "  -h, --help             Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
    
    # If version is "latest", get the latest release version
    if [[ "$VERSION" == "latest" ]]; then
        VERSION=$(get_latest_release)
    fi
    
    [[ -z "$VERSION" ]] && error "Could not determine version to install"
    
    echo "Installing combine-mcp $VERSION..."
    
    # Detect platform
    PLATFORM=$(detect_platform)
    OS=$(echo $PLATFORM | cut -d'-' -f1)
    echo "Detected platform: $PLATFORM"
    
    # Check sudo access
    CAN_USE_SUDO="false"
    if has_sudo; then
        CAN_USE_SUDO="true"
    fi
    
    # Find suitable installation directory
    INSTALL_DIR=$(find_install_dir "$OS" "$CAN_USE_SUDO")
    echo "Selected installation directory: $INSTALL_DIR"
    
    # Download and install binary
    download_binary "$VERSION" "$PLATFORM" "$INSTALL_DIR"
    
    echo "combine-mcp $VERSION has been successfully installed!"
    echo "You can now run it with: combine-mcp"
}

# Run main function with all arguments
main "$@" 
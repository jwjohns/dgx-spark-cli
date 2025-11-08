#!/bin/bash
# DGX Spark CLI Installation Script

set -e

echo "Installing DGX Spark CLI..."
echo "========================"

# Build the binary
echo "Building binary..."
go build -ldflags "-X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" -o dgx ./cmd/dgx

# Determine installation directory
if [ -n "$GOPATH" ]; then
    INSTALL_DIR="$GOPATH/bin"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="/usr/local/bin"
    NEED_SUDO=1
fi

echo "Installing to $INSTALL_DIR..."

# Install binary
if [ "$NEED_SUDO" = "1" ]; then
    sudo cp dgx "$INSTALL_DIR/"
else
    cp dgx "$INSTALL_DIR/"
fi

# Make executable
if [ "$NEED_SUDO" = "1" ]; then
    sudo chmod +x "$INSTALL_DIR/dgx"
else
    chmod +x "$INSTALL_DIR/dgx"
fi

echo ""
echo "✓ DGX Spark CLI installed successfully!"
echo ""
echo "Get started:"
echo "  dgx config set    # Configure your DGX connection"
echo "  dgx status        # Test connection"
echo "  dgx --help        # Show all commands"
echo ""

# Check if installation directory is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "⚠️  Warning: $INSTALL_DIR is not in your PATH"
    echo "   Add this to your ~/.bashrc or ~/.zshrc:"
    echo "   export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
fi

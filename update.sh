#!/bin/bash
# DGX Manager Update Script

set -e

cd "$(dirname "$0")"

echo "Updating DGX Manager..."
echo "======================"

# Pull latest changes
if [ -d .git ]; then
    echo "Pulling latest changes..."
    git pull
fi

# Build
echo "Building..."
go build -ldflags "-X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" -o bin/dgx ./cmd/dgx

# Install
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"
cp bin/dgx "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/dgx"

echo ""
echo "✓ DGX Manager updated successfully!"
echo ""
dgx version

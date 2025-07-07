#!/bin/bash

echo "Building Digital I/O Bank Simulator..."

# Kill any existing server processes to ensure clean restart
echo "Stopping any existing server processes..."
pkill -f "digital-io-server" 2>/dev/null || true
sleep 1

# Ensure configs directory exists
mkdir -p configs

# Get dependencies
go mod tidy

# Build the application
go build -o digital-io-server ./cmd/

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: digital-io-server"
    echo "Note: Any running server has been stopped. Use ./run.sh to start with fresh config."
else
    echo "Build failed!"
    exit 1
fi

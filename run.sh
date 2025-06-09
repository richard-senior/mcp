#!/bin/bash

# Exit on error
set -e

# Navigate to project root (where this script is located)
cd "$(dirname "$0")"

# Build the application first
./build.sh

# Run the application
echo "Starting MCP server..."
./mcp

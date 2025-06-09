#!/bin/bash
set -e

echo "Building MCP application..."
go build -o mcp ./cmd/main.go

echo "Build complete. Binary is at: $(pwd)/mcp"
echo "To use with Amazon Q, make sure your ~/.aws/amazonq/mcp.json points to this binary."

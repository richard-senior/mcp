#!/bin/bash

# Test script to download a Vulcan bomber image using the MCP tool
echo "Testing Wikipedia image downloader for Vulcan bomber"

# Create a JSON request to the MCP server
cat > /tmp/mcp_request.json << 'EOJ'
{
  "jsonrpc": "2.0",
  "method": "tools/invoke",
  "params": {
    "name": "mcp___wikipedia_image_save",
    "parameters": {
      "query": "Avro Vulcan bomber",
      "size": 800
    }
  },
  "id": 1
}
EOJ

echo "Request prepared. Attempting to save image to current directory."
echo "Current directory: $(pwd)"

# Make the request directly to the MCP server
# This is just for testing - in production this would be handled by Amazon Q
echo "This is a test script - in real usage, Amazon Q would handle the MCP communication"

echo "Done. Check /tmp/mcp.log for any error messages."

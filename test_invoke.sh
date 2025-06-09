#!/bin/bash

# Test script to invoke the MCP tool directly
echo "Testing direct invocation of MCP tool"

# Create a JSON request to invoke the wikipedia_image_save tool
cat > /tmp/mcp_invoke.json << 'EOJ'
{
  "jsonrpc": "2.0",
  "method": "tools/invoke",
  "params": {
    "name": "mcp___wikipedia_image_save",
    "parameters": {
      "query": "Avro Vulcan",
      "size": 800
    }
  },
  "id": 2
}
EOJ

# Send the request to the MCP server
echo "Sending request to MCP server..."
cat /tmp/mcp_invoke.json | /Users/richard/mcp/mcp

echo "Done."

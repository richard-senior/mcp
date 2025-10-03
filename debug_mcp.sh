#!/bin/bash
# Debug wrapper for MCP server
echo "=== MCP Debug Wrapper Started ===" >> /tmp/mcp_debug.log
echo "Args: $@" >> /tmp/mcp_debug.log
echo "=== STDIN ===" >> /tmp/mcp_debug.log
tee -a /tmp/mcp_debug.log | /Users/richard/mcp/mcp 2>> /tmp/mcp_debug.log
echo "=== MCP Debug Wrapper Ended ===" >> /tmp/mcp_debug.log

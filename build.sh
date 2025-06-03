#!/bin/bash

function read_properties {
    local search_key="$1"
    local file="${HOME}/.aws/passwords.txt"

    if [ -z "$search_key" ]; then
        echo "Error: No key provided" >&2
        return 1
    fi

    if [ ! -f "$file" ]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi

    # Use -r to prevent backslash escaping
    # Use -d '' to read the entire line including the ending
    while IFS='=' read -r -d $'\n' key value || [ -n "$key" ]; do
        # Skip comments and empty lines
        [[ $key =~ ^#.*$ || -z $key ]] && continue

        # Remove any leading/trailing whitespace
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)

        if [ "$key" = "$search_key" ]; then
            echo "$value"
            return 0
        fi
    done < "$file"

    return 1
}

# Exit on error
set -e

echo "Building MCP CLI application..." >&2

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Change to the project root directory
cd "$DIR"

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the application
go build -o bin/mcp ./cmd/mcp

echo "Build completed successfully. Binary is at bin/mcp" >&2

# Copy the binary to /usr/local/bin and make it executable
echo "Installing MCP binary to /usr/local/bin/mcp..." >&2

# Get the admin password
ADMIN_PASSWORD=$(read_properties "LAPTOP")
if [ -z "$ADMIN_PASSWORD" ]; then
  echo "Warning: Could not retrieve admin password, falling back to sudo" >&2
  
  # Check if we have permission to write to /usr/local/bin directly
  if [ -w /usr/local/bin ]; then
    cp bin/mcp /usr/local/bin/mcp
    chmod +x /usr/local/bin/mcp
  else
    # Use sudo if we don't have direct write permission
    sudo cp bin/mcp /usr/local/bin/mcp
    sudo chmod +x /usr/local/bin/mcp
  fi
else
  # Use the admin password with the 'echo' and 'pipe to sudo -S' approach
  echo "$ADMIN_PASSWORD" | sudo -S cp bin/mcp /usr/local/bin/mcp
  echo "$ADMIN_PASSWORD" | sudo -S chmod +x /usr/local/bin/mcp
fi

echo "Installation completed. MCP is now available system-wide." >&2

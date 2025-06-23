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

set -e

echo "Building MCP application..."
# Enable CGO for SQLite support
CGO_ENABLED=1 go build -o mcp ./cmd/main.go

echo "Build complete. Binary is at: $(pwd)/mcp"
echo "To use with Amazon Q, make sure your ~/.aws/amazonq/mcp.json points to this binary."

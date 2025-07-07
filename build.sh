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

function restore_cache_if_needed {
    local cache_dir="/Users/richard/mcp/_podds/cache"
    local zip_file="/Users/richard/mcp/_podds/podds-cache-data.zip"
    
    echo "Checking cache directory..."
    
    # Check if cache directory exists and has files
    if [ -d "$cache_dir" ] && [ "$(ls -A "$cache_dir" 2>/dev/null)" ]; then
        echo "Cache directory exists and contains files. No restoration needed."
        return 0
    fi
    
    # Check if zip file exists
    if [ ! -f "$zip_file" ]; then
        echo "Warning: Cache directory is empty and zip file not found at $zip_file"
        return 1
    fi
    
    echo "Cache directory is empty. Restoring from $zip_file..."
    
    # Create cache directory if it doesn't exist
    mkdir -p "$cache_dir"
    
    # Unzip the cache data
    cd "/Users/richard/mcp/_podds"
    unzip -q "podds-cache-data.zip" -d "cache/"
    
    if [ $? -eq 0 ]; then
        echo "Cache restored successfully from zip file."
        echo "Restored $(ls -1 "$cache_dir" | wc -l) files to cache directory."
    else
        echo "Error: Failed to restore cache from zip file."
        return 1
    fi
}

set -e

echo "Building MCP application..."

# Restore cache if needed before building
restore_cache_if_needed

# Using pure Go SQLite driver - no CGO required
go build -o mcp ./cmd/main.go

echo "Build complete. Binary is at: $(pwd)/mcp"
echo "To use with Amazon Q, make sure your ~/.aws/amazonq/mcp.json points to this binary."

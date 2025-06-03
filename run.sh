#!/bin/bash

# Exit on error
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Change to the project root directory
cd "$DIR"

# Run the build script
./build.sh

# Function to run a test query and display the results
run_test_query() {
  local query="$1"
  local request_id="$2"
  local description="$3"
  
  echo "Testing: $description"
  echo "Query: $query"
  echo "----------------------------------------"
  echo "{\"query\": \"$query\", \"requestId\": \"$request_id\"}" | ./bin/mcp
  echo "----------------------------------------"
  echo ""
}

# If no arguments are provided, run test queries
if [ $# -eq 0 ]; then
  # Test the calculator
  run_test_query "calculate 2 + 2" "calc-123" "Calculator functionality"
  
  # Test the Google search - use a simpler query without quotes
  run_test_query "googlesearch elvis 3" "search-123" "Google search functionality"
else
  # Run the compiled binary with provided arguments
  ./bin/mcp "$@"
fi

#!/bin/bash

./build.sh

# test.sh - Run all tests in the /Users/richard/mcp/test directory

# Set the working directory to the project root
cd "$(dirname "$0")" || exit 1

# Print header
echo "====================================="
echo "Running all tests in the test directory"
echo "====================================="
echo

# Run the tests
go test -v ./test/...

# Check the exit code
if [ $? -eq 0 ]; then
    echo
    echo "====================================="
    echo "All tests passed!"
    echo "====================================="
    exit 0
else
    echo
    echo "====================================="
    echo "Some tests failed!"
    echo "====================================="
    exit 1
fi

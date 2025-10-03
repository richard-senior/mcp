#!/bin/bash

# Test script for running INITIAL_POSITIONS validation test
echo "Running INITIAL_POSITIONS validation test..."

# Build the project first
./build.sh

# Run the specific test
go test -v ./test -run TestInitialPositionsValidation

echo "Test completed."

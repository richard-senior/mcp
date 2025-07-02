#!/bin/bash

# Parameter tuning test script
# Runs multiple parameter tuning tests

echo "Running Football Data Unit Tests (no HTTP required)"
echo "======================================"

cd /Users/richard/mcp
go test -v ./test

#!/bin/bash

# Parameter tuning test script
# Runs multiple parameter tuning tests

echo "Running Football Data Unit Tests (no HTTP required)"
echo "======================================"

cd /Users/richard/mcp
go test -v ./test -run TestFotmob
go test -v ./test -run TestParseFootballDataCSV
go test -v ./test -run TestValidateInputParameters
go test -v ./test -run TestCacheFilename
go test -v ./test -run TestCurrentSeason

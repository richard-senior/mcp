#!/bin/bash

# Quick Parameter Tuning Test Script
# Runs only the single parameter test without extra checks

cd /Users/richard/mcp

echo "🎯 Running Podds Update with Predictions"
go test ./test -v -count=1 -run TestUpdate

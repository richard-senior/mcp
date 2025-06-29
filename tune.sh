#!/bin/bash

# Quick Parameter Tuning Test Script
# Runs only the single parameter test without extra checks

cd /Users/richard/mcp

echo "🎯 Running parameter tuning test..."
go test ./test -v -run TestTuning

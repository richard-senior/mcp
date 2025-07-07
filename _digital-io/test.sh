#!/bin/bash

echo "Running tests for Digital I/O Bank Simulator..."

# Run all tests
go test -v ./test/

if [ $? -eq 0 ]; then
    echo "All tests passed!"
else
    echo "Some tests failed!"
    exit 1
fi

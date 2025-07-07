#!/bin/bash

echo "Building and running Digital I/O Bank Simulator..."

# Build first
./build.sh

if [ $? -ne 0 ]; then
    echo "Build failed, cannot start server"
    exit 1
fi

./digital-io-server
#!/bin/bash

# Function to cleanup background processes
cleanup() {
    echo "Shutting down all background processes..."
    kill $TTYD1_PID $TTYD2_PID $TTYD3_PID $IO_SERVER_PID 2>/dev/null
    # Also kill by process name as backup
    pkill -f "ttyd -p 768[1-3]" 2>/dev/null
    pkill -f "digital-io-server" 2>/dev/null
    exit 0
}

# Set trap to cleanup on script exit, interrupt, or termination
trap cleanup EXIT INT TERM

# build the io app
echo "rebuilding digital-io app"
cd /Users/richard/mcp/_digital-io
./build.sh

echo "Starting ttyd instance 1..."
ttyd -p 7681 -W bash -c "cd /Users/richard/mcp; q chat;bash" > /tmp/ttyd-7681.log 2>&1 &
TTYD1_PID=$!
echo "Started ttyd on port 7681 (PID: $TTYD1_PID)"

echo "Starting ttyd instance 2..."
ttyd -p 7682 -W bash -c "cd /Users/richard/blank; q chat;bash" > /tmp/ttyd-7682.log 2>&1 &
TTYD2_PID=$!
echo "Started ttyd on port 7682 (PID: $TTYD2_PID)"

echo "Starting ttyd instance 3..."
ttyd -p 7683 -W bash -c "cd /Users/richard/mcp; q chat;bash" > /tmp/ttyd-7683.log 2>&1 &
TTYD3_PID=$!
echo "Started ttyd on port 7683 (PID: $TTYD3_PID)"

echo "Starting digital-io-server..."
# Start the digital-io server in background with output redirection
nohup ./digital-io-server > /tmp/digital-io-server.log 2>&1 &
IO_SERVER_PID=$!
echo "Started digital-io-server (PID: $IO_SERVER_PID)"

echo ""
echo "All processes started successfully!"
echo "Access ttyd instances at:"
echo "  http://localhost:7681"
echo "  http://localhost:7682"
echo "  http://localhost:7683"
echo ""
echo "Press Ctrl+C or close this terminal to stop all processes"

# Open the HTML file in browser
echo "Opening HTML file..."
open /Users/richard/mcp/_documentation/using-llms.html

echo "Script ready - waiting for processes..."
# Keep script running and wait for all background processes
wait
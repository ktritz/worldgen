#!/bin/bash

# Start server in background and capture logs
echo "Starting server with debug logging..."
cd server
nohup go run main.go > server_debug.log 2>&1 &
SERVER_PID=$!
echo "Server started with PID: $SERVER_PID"

# Wait for server to start
echo "Waiting for server to start..."
sleep 3

# Run our test
echo "Running elevation test..."
cd ..
go run generate_test_data.go

# Kill server
echo "Stopping server..."
kill $SERVER_PID

# Show relevant debug output
echo "=== DEBUG OUTPUT FROM SERVER ==="
grep -E "(DEBUG|Continental|Oceanic|Assigning|WARNING|Error)" server/server_debug.log | head -20

echo "=== FULL SERVER LOG ==="
cat server/server_debug.log
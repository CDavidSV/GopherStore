#!/bin/sh

# Restart the server every 2 hours to clear cache
while true; do
  echo "Starting server..."
  timeout 7200 ./server --addr 0.0.0.0:5001 || true
  echo "Restarting server to clear cache..."
  sleep 1
done

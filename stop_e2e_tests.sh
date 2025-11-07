#!/bin/bash

# Stop Kafka container
echo "Stopping Kafka..."
sudo docker compose down

# Find and kill the consumer process
echo "Stopping consumer..."
PROCESS_ID=$(pgrep -f tracker.py)
if [ -n "$PROCESS_ID" ]; then
  kill $PROCESS_ID
  echo "Consumer process stopped."
else
  echo "Consumer process not found."
fi

# Remove virtual environment
if [ -d ".venv" ]; then
  echo "Removing virtual environment..."
  rm -rf .venv
  echo "Virtual environment removed."
fi

# Remove log file
if [ -f "tracker.log" ]; then
    echo "Removing log file..."
    rm tracker.log
    echo "Log file removed."
fi

echo "Cleanup complete."

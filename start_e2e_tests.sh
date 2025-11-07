#!/bin/bash

# Start Kafka container
sudo docker compose up -d

# Wait for Kafka to be ready
echo "Waiting for Kafka to be ready..."
sleep 30

# Create and activate virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run consumer in the background
python3 -u tracker.py > tracker.log 2>&1 &

# Run producer
python3 producer.py

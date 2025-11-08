#!/bin/bash

echo "🔴 Arrêt de l'environnement Kafka..."
docker compose down

echo "🛑 Arrêt du producteur (producer.py)..."
pkill -f producer.py

echo "🛑 Arrêt du consommateur (tracker.py)..."
pkill -f tracker.py

echo "✅ Environnement arrêté."

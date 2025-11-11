#!/bin/bash

echo "🔴 Arrêt de l'environnement Kafka..."
docker compose down

echo "🛑 Arrêt du producteur (producer)..."
pkill -f "bin/producer"

echo "🛑 Arrêt du consommateur (tracker)..."
pkill -f "bin/tracker"

echo "✅ Environnement arrêté."

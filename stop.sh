#!/bin/bash

echo "🔴 Arrêt de l'environnement Kafka..."
docker compose down

echo "🛑 Arrêt du producteur (producer)..."
pkill -f ./producer

echo "🛑 Arrêt du consommateur (tracker)..."
pkill -f ./tracker

echo "✅ Environnement arrêté."

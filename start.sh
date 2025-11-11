#!/bin/bash

echo "🚀 Démarrage de l'environnement Kafka..."
docker compose up -d

echo "⏳ Attente de 30 secondes pour que Kafka démarre complètement..."
sleep 30

echo "🔥 Création du topic 'orders' dans Kafka..."
docker exec kafka kafka-topics --create --topic orders --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1

echo "🔧 Installation des dépendances Go..."
go mod download

echo "🔨 Compilation des programmes Go..."
mkdir -p bin
go build -o bin/producer producer.go
go build -o bin/tracker tracker.go

echo "🟢 Démarrage du consommateur (tracker) en arrière-plan..."
./bin/tracker > tracker.log 2>&1 &

echo "▶️ Démarrage du producteur (producer)..."
./bin/producer

echo "✅ Le producteur a terminé. Le consommateur tourne en arrière-plan."
echo "Pour arrêter l'environnement, exécutez ./stop.sh"

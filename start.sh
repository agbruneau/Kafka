#!/bin/bash

echo "🚀 Démarrage de l'environnement Kafka..."
docker compose up -d

echo "⏳ Attente de 30 secondes pour que Kafka démarre complètement..."
sleep 30

echo "🔥 Création du topic 'orders' dans Kafka..."
docker exec kafka kafka-topics --create --topic orders --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1

echo "🐍 Création de l'environnement virtuel Python..."
if [ ! -d ".venv" ]; then
    python3 -m venv .venv
fi
source .venv/bin/activate

echo "🐍 Installation des dépendances Python..."
pip install -r requirements.txt

echo "🟢 Démarrage du consommateur (tracker.py) en arrière-plan..."
python -u tracker.py > tracker.log &

echo "▶️ Démarrage du producteur (producer.py)..."
python producer.py

echo "✅ Le producteur a terminé. Le consommateur tourne en arrière-plan."
echo "Pour arrêter l'environnement, exécutez ./stop.sh"

#!/bin/bash

echo "🚀 Démarrage de l'environnement Kafka..."
docker compose up -d

echo "⏳ Attente de 30 secondes pour que Kafka démarre complètement..."
sleep 30

echo "🐍 Installation des dépendances Python..."
pip3 install -r requirements.txt

echo "🟢 Démarrage du consommateur (tracker.py) en arrière-plan..."
python3 -u tracker.py > tracker.log &

echo "▶️ Démarrage du producteur (producer.py)..."
python3 producer.py

echo "✅ Le producteur a terminé. Le consommateur tourne en arrière-plan."
echo "Pour arrêter l'environnement, exécutez ./stop.sh"

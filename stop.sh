#!/bin/bash

# Ce script est conçu pour arrêter proprement l'application de démonstration Kafka.
# Il effectue les actions suivantes :
# 1. Arrêt des processus Go : Il recherche et termine les processus 'producer.go' et 'tracker.go'
#    qui pourraient être en cours d'exécution.
# 2. Arrêt des conteneurs Docker : Il arrête et supprime les conteneurs Kafka et Zookeeper
#    définis dans le fichier `docker-compose.yaml`.

# Affiche les commandes exécutées pour un meilleur suivi.
set -x

# Étape 1: Arrêter les processus Go (producer et tracker)
echo "🔴 Arrêt des processus Go (producer et tracker)..."
pkill -f "go run producer.go"
pkill -f "go run tracker.go"

# Étape 2: Arrêter et supprimer les conteneurs Docker
echo "🔴 Arrêt et suppression des conteneurs Docker..."
docker compose down

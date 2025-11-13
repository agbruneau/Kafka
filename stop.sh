#!/bin/bash

# Ce script est conçu pour arrêter proprement l'application de démonstration Kafka.
# Il effectue les actions suivantes :
# 1. Arrêt des processus Go : Il recherche et termine les processus 'producer.go' et 'tracker.go'
#    qui pourraient être en cours d'exécution.
# 2. Arrêt des conteneurs Docker : Il arrête et supprime les conteneurs Kafka et Zookeeper
#    définis dans le fichier `docker-compose.yaml`.

# Affiche les commandes exécutées pour un meilleur suivi.
set -x

# Étape 1: Arrêter proprement les processus Go (producer et tracker)
echo "🔴 Arrêt des processus Go (producer et tracker)..."
echo "   Envoi du signal SIGTERM pour arrêt propre..."

# Envoyer SIGTERM pour permettre le traitement des messages en cours
pkill -TERM -f "go run producer.go"
pkill -TERM -f "go run tracker.go"

# Attendre jusqu'à 10 secondes pour que les processus se terminent proprement
echo "   Attente du traitement des messages en cours (max 10 secondes)..."
for i in {1..10}; do
    if ! pgrep -f "go run producer.go" > /dev/null && ! pgrep -f "go run tracker.go" > /dev/null; then
        echo "   ✅ Tous les processus se sont arrêtés proprement"
        break
    fi
    sleep 1
done

# Si les processus sont toujours actifs après 10 secondes, forcer l'arrêt
if pgrep -f "go run producer.go" > /dev/null || pgrep -f "go run tracker.go" > /dev/null; then
    echo "   ⚠️  Certains processus sont encore actifs - arrêt forcé..."
    pkill -9 -f "go run producer.go"
    pkill -9 -f "go run tracker.go"
fi

# Étape 2: Arrêter et supprimer les conteneurs Docker
echo "🔴 Arrêt et suppression des conteneurs Docker..."
docker compose down

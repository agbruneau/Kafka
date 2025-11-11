#!/bin/bash

# Ce script est responsable du démarrage de l'ensemble de l'application de démonstration Kafka.
# Il exécute les étapes suivantes :
# 1. Démarrage des conteneurs Docker : Lance les services Kafka et Zookeeper en arrière-plan.
# 2. Pause pour l'initialisation : Attend 30 secondes pour s'assurer que Kafka est prêt à accepter des connexions.
# 3. Création du topic Kafka : Crée le topic 'orders' si celui-ci n'existe pas déjà.
# 4. Préparation des applications Go : Télécharge les dépendances et compile les exécutables.
# 5. Lancement du consommateur : Démarre le 'tracker' en arrière-plan pour qu'il écoute les messages.
# 6. Lancement du producteur : Démarre le 'producer' au premier plan, qui commencera à envoyer des messages.

# Affiche les commandes exécutées pour un meilleur suivi.
set -x

# Étape 1: Démarrage des conteneurs Docker
echo "🚀 Démarrage des conteneurs Docker..."
docker compose up -d

# Étape 2: Pause pour l'initialisation
echo "⏳ Attente de 30 secondes pour l'initialisation de Kafka..."
sleep 30

# Étape 3: Création du topic Kafka 'orders'
echo "📝 Création du topic Kafka 'orders'..."
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic orders --partitions 1 --replication-factor 1

# Étape 4: Téléchargement des dépendances Go
echo "📦 Téléchargement des dépendances Go..."
go mod download

# Étape 5: Lancement du consommateur (tracker) en arrière-plan
echo "🟢 Lancement du consommateur (tracker) en arrière-plan..."
go run tracker.go &

# Étape 6: Lancement du producteur (producer) au premier plan
echo "🟢 Lancement du producteur (producer) au premier plan..."
go run producer.go

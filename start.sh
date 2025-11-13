#!/bin/bash

# Ce script est responsable du démarrage de l'ensemble de l'application de démonstration Kafka.
# Il exécute les étapes suivantes :
# 1. Démarrage des conteneurs Docker : Lance les services Kafka et Zookeeper en arrière-plan.
# 2. Pause pour l'initialisation : Attend 30 secondes pour s'assurer que Kafka est prêt à accepter des connexions.
# 3. Création du topic Kafka : Crée le topic 'orders' avec plusieurs partitions pour la scalabilité horizontale.
# 4. Préparation des applications Go : Télécharge les dépendances et compile les exécutables.
# 5. Lancement des consommateurs : Démarre plusieurs instances du 'tracker' en arrière-plan (Competing Consumers pattern).
# 6. Lancement du producteur : Démarre le 'producer' au premier plan, qui commencera à envoyer des messages.

# Affiche les commandes exécutées pour un meilleur suivi.
set -x

# Étape 1: Démarrage des conteneurs Docker
echo "🚀 Démarrage des conteneurs Docker..."
docker compose up -d

# Étape 2: Pause pour l'initialisation
echo "⏳ Attente de 30 secondes pour l'initialisation de Kafka..."
sleep 30

# Étape 3: Création du topic Kafka 'orders' avec plusieurs partitions pour la scalabilité
echo "📝 Création/Configuration du topic Kafka 'orders' avec 3 partitions..."
# Vérifier si le topic existe déjà
if docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list | grep -q "^orders$"; then
    echo "   Le topic 'orders' existe déjà. Augmentation du nombre de partitions à 3..."
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --alter --topic orders --partitions 3 2>/dev/null || {
        echo "   ⚠️  Impossible d'augmenter les partitions. Suppression et recréation du topic..."
        docker exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic orders 2>/dev/null
        docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic orders --partitions 3 --replication-factor 1
    }
else
    echo "   Création du topic 'orders' avec 3 partitions..."
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic orders --partitions 3 --replication-factor 1
fi

# Étape 4: Téléchargement des dépendances Go
echo "📦 Téléchargement des dépendances Go..."
go mod download

# Étape 5: Lancement de plusieurs instances du consommateur (tracker) en arrière-plan
# Pattern Competing Consumers : plusieurs instances dans le même consumer group
NUM_INSTANCES=3
echo "🟢 Lancement de $NUM_INSTANCES instances du consommateur (tracker) en arrière-plan..."
echo "   Pattern: Competing Consumers (scalabilité horizontale)"
for i in $(seq 1 $NUM_INSTANCES); do
    echo "   Instance $i/$NUM_INSTANCES..."
    TRACKER_INSTANCE_ID="instance-$i" go run tracker.go order.go &
    sleep 1  # Petit délai pour éviter les conflits d'initialisation
done
echo "   ✅ $NUM_INSTANCES instances lancées dans le consumer group 'order-tracker'"

# Étape 6: Lancement du producteur (producer) au premier plan
echo "🟢 Lancement du producteur (producer) au premier plan..."
go run producer.go order.go

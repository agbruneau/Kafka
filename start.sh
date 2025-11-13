#!/bin/bash

# ==============================================================================
# SCRIPT DE DÉMARRAGE DE L'APPLICATION KAFKA DEMO
# ==============================================================================
#
# Ce script orchestre le démarrage complet de l'environnement de démonstration.
# Il exécute les étapes suivantes dans un ordre précis pour garantir que
# tous les composants sont prêts et connectés correctement.
#
# Étapes exécutées :
# 1. Démarrage des conteneurs Docker : Lance le service Kafka en arrière-plan
#    en utilisant la configuration de `docker-compose.yaml`.
# 2. Pause d'initialisation : Attend un temps défini (30 secondes) pour
#    s'assurer que le broker Kafka est entièrement initialisé et prêt à
#    accepter des connexions et des commandes.
# 3. Création du topic Kafka : Crée le topic 'orders', qui est le canal de
#    communication entre le producteur et le consommateur.
# 4. Installation des dépendances Go : Exécute `go mod download` pour
#    télécharger les bibliothèques nécessaires (client Kafka, UUID).
# 5. Lancement du consommateur (`tracker`) : Démarre le consommateur en
#    arrière-plan. Il commencera immédiatement à écouter les messages
#    sur le topic 'orders'.
# 6. Lancement du producteur (`producer`) : Démarre le producteur au
#    premier plan. Il commencera à générer et envoyer des messages.
#    Le script se terminera lorsque le producteur sera arrêté (Ctrl+C).
#
# ------------------------------------------------------------------------------

# Active le mode "verbose" pour afficher chaque commande avant son exécution.
# Utile pour le débogage.
set -x

# Étape 1: Démarrage des conteneurs Docker
echo "🚀 Démarrage des conteneurs Docker (Kafka)..."
sudo docker compose up -d

# Étape 2: Attente active de la disponibilité de Kafka
echo "⏳ Attente de la disponibilité du broker Kafka..."
until docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; do
  echo "Kafka n'est pas encore prêt, nouvelle tentative dans 5 secondes..."
  sleep 5
done
echo "✅ Kafka est prêt !"

# Étape 3: Création du topic Kafka 'orders'
# Cette commande est idempotente ; elle ne fera rien si le topic existe déjà.
echo "📝 Création du topic Kafka 'orders' (s'il n'existe pas)..."
docker exec kafka kafka-topics \
  --bootstrap-server localhost:9092 \
  --create \
  --topic orders \
  --partitions 1 \
  --replication-factor 1

# Étape 4: Téléchargement des dépendances Go
echo "📦 Téléchargement des dépendances Go via 'go mod download'..."
go mod download

# Étape 5: Lancement du consommateur (tracker) en arrière-plan
# Le `&` à la fin de la commande le fait tourner en tâche de fond.
# Les logs du tracker seront visibles dans les fichiers tracker.log et tracker.events.
echo "🟢 Lancement du consommateur (tracker) en arrière-plan..."
go run tracker.go order.go &

# Étape 6: Lancement du producteur (producer) au premier plan
# Le script attendra ici jusqu'à ce que le producteur soit manuellement arrêté.
echo "🟢 Lancement du producteur (producer) au premier plan..."
go run producer.go order.go

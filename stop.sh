#!/bin/bash

# ==============================================================================
# SCRIPT D'ARRÊT PROPRE DE L'APPLICATION KAFKA DEMO
# ==============================================================================
#
# Ce script est conçu pour arrêter proprement tous les composants de l'application.
# Il suit une approche en plusieurs étapes pour s'assurer que les données en
# transit sont traitées avant l'arrêt complet.
#
# Étapes exécutées :
# 1. Arrêt des processus Go :
#    a. Envoi d'un signal SIGTERM : Ce signal demande aux processus Go de
#       s'arrêter proprement. Le producteur videra son tampon et le
#       consommateur terminera de traiter le message en cours.
#    b. Période de grâce : Le script attend jusqu'à 10 secondes pour laisser
#       le temps aux applications de se terminer d'elles-mêmes.
#    c. Arrêt forcé (si nécessaire) : Si les processus sont toujours actifs
#       après le délai, un signal SIGKILL est envoyé pour les forcer à
#       s'arrêter. C'est une mesure de sécurité.
# 2. Arrêt des conteneurs Docker : Une fois les applications Go terminées,
#    `docker compose down` est appelé pour arrêter et supprimer les conteneurs
#    Kafka.
#
# ------------------------------------------------------------------------------

# Active le mode "verbose" pour afficher chaque commande.
set -x

# Étape 1: Arrêter proprement les processus Go (producer et tracker)
echo "🔴 Arrêt des processus applicatifs Go..."
echo "   1. Envoi du signal SIGTERM pour un arrêt gracieux..."

# `pkill -f` recherche le nom du processus dans la ligne de commande complète.
# Le signal SIGTERM (-TERM) est intercepté par nos applications Go pour
# déclencher la logique d'arrêt propre.
pkill -TERM -f "go run producer.go order.go"
pkill -TERM -f "go run tracker.go order.go"

# Période de grâce pour permettre aux processus de s'arrêter d'eux-mêmes.
echo "   2. Attente de 10 secondes pour le traitement des messages en cours..."
for i in {1..10}; do
    # `pgrep -f` vérifie si les processus existent toujours.
    if ! pgrep -f "go run producer.go order.go" && ! pgrep -f "go run tracker.go order.go"; then
        echo "   ✅ Les processus Go se sont arrêtés proprement."
        break
    fi
    sleep 1
    # Indicateur visuel pour montrer que le script attend.
    echo -n "."
done
echo "" # Saut de ligne après les points.

# Si, après 10 secondes, les processus sont toujours là, on force l'arrêt.
if pgrep -f "go run producer.go order.go" || pgrep -f "go run tracker.go order.go"; then
    echo "   ⚠️  Certains processus sont toujours actifs. Arrêt forcé (SIGKILL)..."
    pkill -9 -f "go run producer.go order.go"
    pkill -9 -f "go run tracker.go order.go"
fi

# Étape 2: Arrêter et supprimer les conteneurs Docker
echo "🔴 Arrêt et suppression des conteneurs Docker..."
sudo docker compose down

echo "✅ L'environnement a été complètement arrêté."

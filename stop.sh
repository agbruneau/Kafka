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

# Obtenir le répertoire du script
script_dir=$(dirname "$0")

# Étape 1: Arrêter proprement les processus Go (producer et tracker)
echo "🔴 Arrêt des processus applicatifs Go..."
echo "   1. Envoi du signal SIGTERM pour un arrêt gracieux..."

# Vérifier si les fichiers PID existent avant de les lire
if [ -f "$script_dir/producer.pid" ] && [ -f "$script_dir/tracker.pid" ]; then
    producer_pid=$(cat "$script_dir/producer.pid")
    tracker_pid=$(cat "$script_dir/tracker.pid")

    # Tuer les processus en utilisant les PIDs
    kill -TERM $producer_pid
    kill -TERM $tracker_pid

    # Période de grâce pour permettre aux processus de s'arrêter d'eux-mêmes.
    echo "   2. Attente de 10 secondes pour le traitement des messages en cours..."
    for i in {1..10}; do
        if ! kill -0 $producer_pid 2>/dev/null && ! kill -0 $tracker_pid 2>/dev/null; then
            echo "   ✅ Les processus Go se sont arrêtés proprement."
            break
        fi
        sleep 1
        echo -n "."
    done
    echo ""

    # Si, après 10 secondes, les processus sont toujours là, on force l'arrêt.
    if kill -0 $producer_pid 2>/dev/null || kill -0 $tracker_pid 2>/dev/null; then
        echo "   ⚠️  Certains processus sont toujours actifs. Arrêt forcé (SIGKILL)..."
        kill -9 $producer_pid
        kill -9 $tracker_pid
    fi

    # Nettoyer les fichiers PID
    rm -f "$script_dir/producer.pid" "$script_dir/tracker.pid"
else
    echo "   ⚠️ Fichiers PID non trouvés. Tentative d'arrêt par pkill..."
    pkill -TERM -f "go run producer.go order.go"
    pkill -TERM -f "go run tracker.go order.go"
fi

# Étape 2: Arrêter et supprimer les conteneurs Docker
echo "🔴 Arrêt et suppression des conteneurs Docker..."
sudo docker compose down

echo "✅ L'environnement a été complètement arrêté."

#!/bin/bash

# ==============================================================================
# SCRIPT D'ANALYSE DES LOGS D'OBSERVABILITÉ (`tracker.log`)
# ==============================================================================
#
# Ce script fournit une analyse de base du fichier de log `tracker.log`,
# qui contient les logs système structurés au format JSON.
#
# Il extrait des informations clés telles que :
# - Le nombre total d'entrées de log.
# - La répartition des logs par niveau (INFO, ERROR).
# - Le nombre de commandes traitées avec succès.
# - Un résumé des erreurs détectées.
#
# Si l'outil `jq` (un processeur JSON en ligne de commande) est installé,
# le script fournit également des statistiques plus avancées :
# - Le montant total des commandes.
# - Le montant moyen par commande.
# - Le top 5 des clients par nombre de commandes.
#
# Utilisation :
# 1. Rendez le script exécutable : `chmod +x analyze_logs.sh`
# 2. Exécutez-le : `./analyze_logs.sh`
#
# ------------------------------------------------------------------------------

LOG_FILE="tracker.log"
EVENTS_FILE="tracker.events"

# Vérifie si le fichier de log principal existe.
if [ ! -f "$LOG_FILE" ]; then
    echo "❌ Le fichier de log '$LOG_FILE' est introuvable."
    echo "   Veuillez lancer l'application (./start.sh) pour le générer."
    exit 1
fi

echo "📊 ANALYSE DES LOGS - $LOG_FILE"
echo "================================================="
echo ""

# --- Statistiques Générales ---
echo "📈 STATISTIQUES GÉNÉRALES"
echo "-------------------------------------------------"
TOTAL_LOGS=$(wc -l < "$LOG_FILE")
echo "   - Nombre total d'entrées de log : $TOTAL_LOGS"

# Répartition par niveau de log en utilisant `grep` et `awk`.
INFO_COUNT=$(grep -c '"level":"INFO"' "$LOG_FILE")
ERROR_COUNT=$(grep -c '"level":"ERROR"' "$LOG_FILE")
echo "   - Entrées de niveau INFO        : $INFO_COUNT"
echo "   - Entrées de niveau ERROR       : $ERROR_COUNT"
echo ""

# --- Analyse des Événements (`tracker.events`) ---
if [ -f "$EVENTS_FILE" ]; then
    echo "📋 ANALYSE DES ÉVÉNEMENTS - $EVENTS_FILE"
    echo "-------------------------------------------------"
    TOTAL_EVENTS=$(wc -l < "$EVENTS_FILE")
    PROCESSED_EVENTS=$(grep -c '"deserialized":true' "$EVENTS_FILE")
    FAILED_EVENTS=$(grep -c '"deserialized":false' "$EVENTS_FILE")
    echo "   - Nombre total de messages reçus : $TOTAL_EVENTS"
    echo "   - Messages traités avec succès   : $PROCESSED_EVENTS"
    echo "   - Échecs de désérialisation      : $FAILED_EVENTS"
    echo ""
fi


# --- Analyse des Erreurs ---
echo "🚨 ANALYSE DES ERREURS"
echo "-------------------------------------------------"
if [ "$ERROR_COUNT" -gt 0 ]; then
    echo "   - ❌ $ERROR_COUNT erreur(s) détectée(s) dans '$LOG_FILE'."
    echo "   - Dernières erreurs :"
    # Affiche les erreurs de manière lisible, avec `jq` si possible.
    if command -v jq &> /dev/null; then
        grep '"level":"ERROR"' "$LOG_FILE" | tail -5 | jq -r '"     [\(.timestamp)] \(.message) | Détails: \(.error // "N/A")"'
    else
        grep '"level":"ERROR"' "$LOG_FILE" | tail -5
    fi
else
    echo "   - ✅ Aucune erreur détectée."
fi
echo ""


# --- Statistiques Métier (nécessite `jq`) ---
if command -v jq &> /dev/null; then
    echo "💼 STATISTIQUES MÉTIER (depuis '$EVENTS_FILE')"
    echo "-------------------------------------------------"
    
    # Calcule le montant total et moyen à partir des événements valides.
    TOTAL_AMOUNT=$(grep '"deserialized":true' "$EVENTS_FILE" | jq -r '.order_full.total' | awk '{sum+=$1} END {printf "%.2f", sum}')
    AVG_AMOUNT=$(grep '"deserialized":true' "$EVENTS_FILE" | jq -r '.order_full.total' | awk '{sum+=$1; count++} END {if(count>0) printf "%.2f", sum/count; else print "0.00"}')
    echo "   - Chiffre d'affaires total : ${TOTAL_AMOUNT:-0.00} EUR"
    echo "   - Panier moyen             : ${AVG_AMOUNT:-0.00} EUR"
    echo ""

    # Identifie le top 5 des clients.
    echo "   - Top 5 des clients par commandes :"
    grep '"deserialized":true' "$EVENTS_FILE" | jq -r '.order_full.customer_info.customer_id' | sort | uniq -c | sort -rn | head -5 | awk '{printf "     - %-20s : %d commande(s)\n", $2, $1}'
    echo ""
else
    echo "ℹ️  Pour des statistiques métier (chiffre d'affaires, top clients), veuillez installer 'jq'."
    echo "    Exemple: sudo apt-get install jq"
    echo ""
fi


# --- Dernières Activités ---
echo "📝 DERNIÈRES ACTIVITÉS DANS '$LOG_FILE'"
echo "-------------------------------------------------"
# Affiche les 5 dernières lignes de log de manière formatée.
if command -v jq &> /dev/null; then
    tail -5 "$LOG_FILE" | jq -r '"   [\(.timestamp)] [\(.level)] \(.message)"'
else
    tail -5 "$LOG_FILE"
fi
echo ""
echo "================================================="
echo "💡 Pour une analyse manuelle, utilisez des outils comme 'jq', 'grep' et 'awk'."
echo "   Ex: jq '. | select(.level == \"ERROR\")' tracker.log"
echo "   Ex: jq '. | select(.deserialized == true) | .order_full' tracker.events"

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

# Définition des couleurs pour une sortie plus lisible
BLUE="\e[34m"
GREEN="\e[32m"
RED="\e[31m"
YELLOW="\e[33m"
RESET="\e[0m"

# Bannière
echo -e "${BLUE}=================================================${RESET}"
echo -e "${BLUE}📊       RAPPORT D'ANALYSE DES LOGS        📊${RESET}"
echo -e "${BLUE}=================================================${RESET}"
echo ""

# --- Statistiques Générales ---
echo -e "${GREEN}📈 STATISTIQUES GÉNÉRALES${RESET}"
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
    echo -e "${GREEN}📋 ANALYSE DES ÉVÉNEMENTS - $EVENTS_FILE${RESET}"
    echo "-------------------------------------------------"
    TOTAL_EVENTS=$(wc -l < "$EVENTS_FILE")
    PROCESSED_EVENTS=$(grep -c '"deserialized":true' "$EVENTS_FILE")
    FAILED_EVENTS=$(grep -c '"deserialized":false' "$EVENTS_FILE")
    echo "   - Nombre total de messages reçus : $TOTAL_EVENTS"
    echo "   - Messages traités avec succès   : $PROCESSED_EVENTS"
    echo "   - Échecs de désérialisation      : $FAILED_EVENTS"
    echo ""
fi


# --- Analyse de Performance ---
if command -v jq &> /dev/null; then
    echo -e "${GREEN}🚀 ANALYSE DE PERFORMANCE${RESET}"
    echo "-------------------------------------------------"
    # Extrait les dernières métriques périodiques depuis tracker.log
    LAST_METRICS_LOG=$(grep '"Métriques système périodiques"' "$LOG_FILE" | tail -1)
    if [ -n "$LAST_METRICS_LOG" ]; then
        MSG_PER_SEC=$(echo "$LAST_METRICS_LOG" | jq -r '.metadata.messages_per_second')
        SUCCESS_RATE=$(echo "$LAST_METRICS_LOG" | jq -r '.metadata.success_rate_percent')
        echo "   - Dernier débit rapporté (tracker) : $MSG_PER_SEC msg/s"
        echo "   - Dernier taux de succès (tracker)  : $SUCCESS_RATE %"
    else
        echo "   - Aucune métrique de performance périodique trouvée dans '$LOG_FILE'."
    fi

    # Calcule le débit moyen global basé sur les timestamps de tracker.events
    if [ -f "$EVENTS_FILE" ] && [ "$(wc -l < "$EVENTS_FILE")" -gt 1 ]; then
        FIRST_TS=$(head -1 "$EVENTS_FILE" | jq -r '.timestamp')
        LAST_TS=$(tail -1 "$EVENTS_FILE" | jq -r '.timestamp')

        # `date` sur Linux peut parser le format ISO 8601 directement.
        START_SECONDS=$(date -d "$FIRST_TS" +%s 2>/dev/null || date -jf "%Y-%m-%dT%H:%M:%SZ" "$FIRST_TS" +%s) # macOS fallback
        END_SECONDS=$(date -d "$LAST_TS" +%s 2>/dev/null || date -jf "%Y-%m-%dT%H:%M:%SZ" "$LAST_TS" +%s) # macOS fallback

        DURATION=$((END_SECONDS - START_SECONDS))
        TOTAL_EVENTS=$(wc -l < "$EVENTS_FILE")

        if [ "$DURATION" -gt 0 ]; then
            AVG_THROUGHPUT=$(awk "BEGIN {printf \"%.2f\", $TOTAL_EVENTS / $DURATION}")
            echo "   - Débit moyen global (events)      : $AVG_THROUGHPUT msg/s sur $DURATION s"
        else
            echo "   - Débit moyen global (events)      : N/A (durée de traitement trop courte)"
        fi
    else
        echo "   - Pas assez de données dans '$EVENTS_FILE' pour calculer le débit global."
    fi
    echo ""
fi


# --- Analyse des Erreurs ---
echo -e "${RED}🚨 ANALYSE DES ERREURS${RESET}"
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

    if [ "$FAILED_EVENTS" -gt 0 ]; then
        echo ""
        echo "   - 🔍 Examen des messages ayant échoué à la désérialisation :"
        grep '"deserialized":false' "$EVENTS_FILE" | tail -5 | jq -r '"     [HORODATAGE: \(.timestamp)] [OFFSET: \(.kafka_offset)]\n       MESSAGE BRUT: \(.raw_message)\n       ERREUR: \(.error)\n"'
    fi
else
    echo "   - ✅ Aucune erreur détectée."
fi
echo ""


# --- Statistiques Métier (nécessite `jq`) ---
if command -v jq &> /dev/null; then
    echo -e "${GREEN}💼 STATISTIQUES MÉTIER (depuis '$EVENTS_FILE')${RESET}"
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

    # --- Statistiques Métier Détaillées ---
    echo "   --- Statistiques Produits ---"

    # Top 5 des produits par quantité vendue
    echo "   - Top 5 des produits par quantité vendue :"
    grep '"deserialized":true' "$EVENTS_FILE" | jq -r '.order_full.items[] | "\(.item_name) \(.quantity)"' | \
    awk '{arr[$1]+=$2} END {for (i in arr) print arr[i], i}' | \
    sort -rn | head -5 | awk '{printf "     - %-20s : %d unités\n", $2, $1}'
    echo ""

    # Top 5 des produits par chiffre d'affaires
    echo "   - Top 5 des produits par chiffre d'affaires :"
    grep '"deserialized":true' "$EVENTS_FILE" | jq -r '.order_full.items[] | "\(.item_name) \(.total_price)"' | \
    awk '{arr[$1]+=$2} END {for (i in arr) print arr[i], i}' | \
    sort -rn | head -5 | awk '{printf "     - %-20s : %.2f EUR\n", $2, $1}'
    echo ""

    echo "   --- Statistiques Paiements ---"

    # Répartition des méthodes de paiement
    echo "   - Répartition des méthodes de paiement :"
    grep '"deserialized":true' "$EVENTS_FILE" | jq -r '.order_full.payment_method' | \
    sort | uniq -c | sort -rn | \
    awk '{printf "     - %-20s : %d transaction(s)\n", $2, $1}'
    echo ""
else
    echo "ℹ️  Pour des statistiques métier (chiffre d'affaires, top clients), veuillez installer 'jq'."
    echo "    Exemple: sudo apt-get install jq"
    echo ""
fi


# --- Dernières Activités ---
echo -e "${GREEN}📝 DERNIÈRES ACTIVITÉS DANS '$LOG_FILE'${RESET}"
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

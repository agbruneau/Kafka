#!/bin/bash

# Script d'analyse des logs tracker.log
# Ce script fournit des exemples d'analyse des logs structurés JSON

LOG_FILE="tracker.log"

if [ ! -f "$LOG_FILE" ]; then
    echo "❌ Le fichier $LOG_FILE n'existe pas."
    echo "   Assurez-vous que le tracker a été exécuté au moins une fois."
    exit 1
fi

echo "📊 ANALYSE DES LOGS - tracker.log"
echo "=================================="
echo ""

# Compter le nombre total de logs
TOTAL=$(wc -l < "$LOG_FILE")
echo "📈 Nombre total d'entrées de log: $TOTAL"
echo ""

# Compter par niveau
echo "📊 Répartition par niveau de log:"
echo "-----------------------------------"
grep -o '"level":"[^"]*"' "$LOG_FILE" | sort | uniq -c | sed 's/"level":"//g' | sed 's/"//g' | awk '{printf "   %-10s: %d entrées\n", $2, $1}'
echo ""

# Compter les commandes traitées
ORDERS=$(grep -c '"message":"Commande reçue et traitée"' "$LOG_FILE" 2>/dev/null || echo "0")
echo "📦 Commandes traitées: $ORDERS"
echo ""

# Afficher les erreurs
ERRORS=$(grep -c '"level":"ERROR"' "$LOG_FILE" 2>/dev/null || echo "0")
if [ "$ERRORS" -gt 0 ]; then
    echo "❌ Erreurs détectées: $ERRORS"
    echo "   Dernières erreurs:"
    grep '"level":"ERROR"' "$LOG_FILE" | tail -5 | jq -r '"   [\(.timestamp)] \(.message) - \(.error // "N/A")"' 2>/dev/null || \
    grep '"level":"ERROR"' "$LOG_FILE" | tail -5
    echo ""
else
    echo "✅ Aucune erreur détectée"
    echo ""
fi

# Statistiques sur les commandes (si jq est disponible)
if command -v jq &> /dev/null; then
    echo "💰 Statistiques financières:"
    echo "----------------------------"
    TOTAL_AMOUNT=$(grep '"message":"Commande reçue et traitée"' "$LOG_FILE" | jq -r '.metadata.total' | awk '{sum+=$1} END {printf "%.2f", sum}')
    AVG_AMOUNT=$(grep '"message":"Commande reçue et traitée"' "$LOG_FILE" | jq -r '.metadata.total' | awk '{sum+=$1; count++} END {if(count>0) printf "%.2f", sum/count; else printf "0.00"}')
    echo "   Total des commandes: ${TOTAL_AMOUNT} EUR"
    echo "   Montant moyen: ${AVG_AMOUNT} EUR"
    echo ""
    
    echo "👥 Top 5 clients:"
    echo "----------------"
    grep '"message":"Commande reçue et traitée"' "$LOG_FILE" | jq -r '.metadata.customer_id' | sort | uniq -c | sort -rn | head -5 | awk '{printf "   %-20s: %d commande(s)\n", $2, $1}'
    echo ""
fi

# Afficher les dernières entrées
echo "📝 Dernières 5 entrées de log:"
echo "-------------------------------"
if command -v jq &> /dev/null; then
    tail -5 "$LOG_FILE" | jq -r '"   [\(.timestamp)] [\(.level)] \(.message)"'
else
    tail -5 "$LOG_FILE"
fi
echo ""

echo "💡 Pour une analyse plus approfondie, utilisez:"
echo "   - jq pour filtrer et analyser les logs JSON"
echo "   - grep pour rechercher des patterns spécifiques"
echo "   - awk pour des calculs personnalisés"
echo ""
echo "   Exemples:"
echo "   # Toutes les commandes d'un client spécifique:"
echo "   grep 'client01' $LOG_FILE | jq"
echo ""
echo "   # Commandes avec un montant supérieur à 50 EUR:"
echo "   grep 'Commande reçue' $LOG_FILE | jq 'select(.metadata.total > 50)'"
echo ""
echo "   # Erreurs avec détails:"
echo "   grep '\"level\":\"ERROR\"' $LOG_FILE | jq"


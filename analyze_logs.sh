#!/bin/bash

# ==============================================================================
# SCRIPT D'ANALYSE DES LOGS D'OBSERVABILITÉ (`tracker.log`) - MODE CONTINU
# ==============================================================================
#
# Ce script exécute le programme Go `analyze_logs.go` qui fournit une analyse
# en temps réel du fichier de log `tracker.log`, qui contient les logs système
# structurés au format JSON. L'affichage est rafraîchi toutes les 2 secondes.
#
# Le programme Go analyse :
# - Les logs système (`tracker.log`) : statistiques générales, erreurs, métriques
# - Les événements (`tracker.events`) : messages reçus, statistiques métier
#
# Utilisation :
# 1. Rendez le script exécutable : `chmod +x analyze_logs.sh`
# 2. Exécutez-le : `./analyze_logs.sh`
# 3. Appuyez sur CTRL+C pour quitter.
#
# ------------------------------------------------------------------------------

# Exécuter le programme Go
# Le script s'exécute depuis le répertoire du projet où se trouvent les fichiers de log
go run analyze_logs.go

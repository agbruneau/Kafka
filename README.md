# Projet de Démonstration Kafka avec Golang

Ce projet est une démonstration d'un système de messagerie basé sur Apache Kafka, entièrement conteneurisé avec Docker. Il illustre un cas d'utilisation simple mais fondamental : un producteur (`producer.go`) qui génère des messages et les envoie à un topic Kafka, et un consommateur (`tracker.go`) qui s'abonne à ce topic pour recevoir et traiter ces messages en temps réel.

## Architecture

L'architecture de ce projet est simple et se compose des éléments suivants :

-   **Apache Kafka** : Le cœur du système, agissant comme un broker de messages. Il est responsable de la réception, du stockage et de la distribution des messages.
-   **Producteur (`producer.go`)** : Un programme Go qui simule la création de commandes. Il génère des messages au format JSON et les envoie au topic Kafka `orders`.
-   **Consommateur (`tracker.go`)** : Un autre programme Go qui s'abonne au topic `orders`. Il écoute en continu les nouveaux messages, les désérialise et affiche leur contenu.
-   **Docker et Docker Compose** : L'ensemble de l'environnement, y compris Kafka et ses dépendances comme Zookeeper, est géré via Docker Compose, garantissant une configuration portable et reproductible.

## Prérequis

Pour exécuter ce projet, vous devez disposer des outils suivants :

-   **Docker** et **Docker Compose**
-   **Go 1.21 ou supérieur**

## Démarrage et Arrêt

### Avec les Scripts

La manière la plus simple de lancer l'application est d'utiliser les scripts fournis :

-   **Pour démarrer** :
    ```bash
    ./start.sh
    ```
    Ce script orchestre le démarrage des conteneurs Docker, la création du topic Kafka nécessaire, la compilation des programmes Go et leur exécution.

-   **Pour arrêter** :
    ```bash
    ./stop.sh
    ```
    Ce script arrête proprement les programmes Go et supprime les conteneurs Docker.

### Manuellement (Sans les Scripts)

Si vous préférez exécuter chaque composant séparément, suivez ces étapes :

1.  **Démarrer l'environnement Docker** :
    ```bash
    docker compose up -d
    ```

2.  **Attendre l'initialisation de Kafka** :
    Après avoir lancé les conteneurs, attendez environ 30 secondes pour que Kafka soit pleinement opérationnel.

3.  **Créer le topic Kafka** :
    ```bash
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic orders --partitions 1 --replication-factor 1
    ```

4.  **Lancer le consommateur** :
    Ouvrez un terminal et exécutez :
    ```bash
    go run tracker.go order.go
    ```

5.  **Lancer le producteur** :
    Ouvrez un second terminal et exécutez :
    ```bash
    go run producer.go order.go
    ```

## Observabilité et Logging

Le système de tracking (`tracker.go`) génère des logs structurés au format JSON dans le fichier `tracker.log`. Ces logs permettent une observabilité complète du système.

### Format des Logs

Les logs sont écrits au format JSON avec les champs suivants :
- `timestamp` : Date et heure de l'événement (RFC3339)
- `level` : Niveau de log (DEBUG, INFO, WARN, ERROR)
- `message` : Message descriptif de l'événement
- `service` : Nom du service (order-tracker)
- `order_id` : ID de la commande (si applicable)
- `sequence` : Numéro de séquence (si applicable)
- `correlation_id` : ID de corrélation pour le suivi
- `metadata` : Métadonnées contextuelles supplémentaires, incluant :
  - **`raw_message`** : Le message brut JSON tel que reçu de Kafka (pour traçabilité complète)
  - **`order_full`** : La structure Order complète sérialisée en JSON (pour analyse détaillée)
  - **`kafka`** : Métadonnées Kafka (topic, partition, offset, key, timestamp)
  - Informations de la commande (status, total, currency, customer, items, inventory_status, etc.)

### Analyse des Logs

Un script d'analyse est fourni pour exploiter les logs :

```bash
chmod +x analyze_logs.sh
./analyze_logs.sh
```

Ce script affiche :
- Statistiques générales (nombre total de logs, répartition par niveau)
- Nombre de commandes traitées
- Détection d'erreurs
- Statistiques financières (si `jq` est installé)
- Top clients
- Dernières entrées de log

### Exemples d'Analyse Manuelle

Avec `jq` (recommandé pour l'analyse JSON) :

```bash
# Toutes les commandes d'un client spécifique
grep 'client01' tracker.log | jq

# Commandes avec un montant supérieur à 50 EUR
grep 'Commande reçue' tracker.log | jq 'select(.metadata.total > 50)'

# Erreurs avec détails
grep '"level":"ERROR"' tracker.log | jq

# Compter les commandes par statut
grep 'Commande reçue' tracker.log | jq -r '.metadata.status' | sort | uniq -c

# Montant total des commandes
grep 'Commande reçue' tracker.log | jq -r '.metadata.total' | awk '{sum+=$1} END {print sum}'

# Extraire le message brut d'une commande spécifique
grep 'Commande reçue' tracker.log | jq -r 'select(.order_id == "votre-order-id") | .metadata.raw_message'

# Extraire la structure complète d'une commande
grep 'Commande reçue' tracker.log | jq 'select(.order_id == "votre-order-id") | .metadata.order_full'

# Analyser les métadonnées Kafka (offset, partition, etc.)
grep 'Commande reçue' tracker.log | jq '.metadata.kafka'

# Reconstruire une commande complète depuis les logs
grep 'Commande reçue' tracker.log | jq 'select(.order_id == "votre-order-id") | .metadata.order_full' | jq
```

Sans `jq` :

```bash
# Compter les erreurs
grep -c '"level":"ERROR"' tracker.log

# Afficher les dernières erreurs
grep '"level":"ERROR"' tracker.log | tail -5

# Compter les commandes traitées
grep -c '"message":"Commande reçue et traitée"' tracker.log
```

## Commandes Utiles pour Kafka

Pour interagir avec Kafka et observer le système, vous pouvez utiliser ces commandes.

-   **Lister les topics** :
    ```bash
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
    ```

-   **Décrire un topic** :
    ```bash
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic orders
    ```

-   **Consommer les messages depuis le terminal** :
    Une excellente façon de déboguer ou de visualiser le flux de messages en temps réel.
    ```bash
    docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
    ```

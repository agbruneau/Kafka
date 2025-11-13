# Projet de Démonstration Kafka avec Golang

Ce projet est une démonstration d'un système de messagerie basé sur Apache Kafka, entièrement conteneurisé avec Docker. Il illustre un cas d'utilisation simple mais fondamental : un producteur (`producer.go`) qui génère des messages et les envoie à un topic Kafka, et un consommateur (`tracker.go`) qui s'abonne à ce topic pour recevoir et traiter ces messages en temps réel.

## Architecture

L'architecture de ce projet est simple et se compose des éléments suivants :

-   **Apache Kafka** : Le cœur du système, agissant comme un broker de messages. Il est responsable de la réception, du stockage et de la distribution des messages.
-   **Producteur (`producer.go`)** : Un programme Go qui simule la création de commandes. Il génère des messages au format JSON et les envoie au topic Kafka `orders`.
-   **Consommateur (`tracker.go`)** : Un autre programme Go qui s'abonne au topic `orders`. Il écoute en continu les nouveaux messages, les désérialise et affiche leur contenu. **Plusieurs instances peuvent être déployées pour la scalabilité horizontale (pattern Competing Consumers)**.
-   **Docker et Docker Compose** : L'ensemble de l'environnement, y compris Kafka et ses dépendances comme Zookeeper, est géré via Docker Compose, garantissant une configuration portable et reproductible.

### Scalabilité Horizontale (Competing Consumers)

Le projet implémente le pattern **Competing Consumers** pour permettre la scalabilité horizontale :

-   **Plusieurs partitions** : Le topic `orders` est configuré avec 3 partitions (au lieu d'une seule) pour permettre le traitement parallèle.
-   **Plusieurs instances** : Par défaut, 3 instances de `tracker.go` sont lancées dans le même consumer group (`order-tracker`).
-   **Répartition automatique** : Kafka distribue automatiquement les partitions entre les instances disponibles.
-   **Bénéfices** :
    - **Scalabilité horizontale** : Ajoutez ou retirez des instances selon la charge
    - **Haute disponibilité** : Si une instance tombe, les autres continuent le traitement
    - **Traitement parallèle** : Plusieurs partitions sont traitées simultanément
    - **Répartition équilibrée** : Kafka équilibre automatiquement la charge entre les instances

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
    Ce script orchestre le démarrage des conteneurs Docker, la création du topic Kafka avec 3 partitions, la compilation des programmes Go et le lancement de 3 instances du consommateur (pattern Competing Consumers).

-   **Pour arrêter** :
    ```bash
    ./stop.sh
    ```
    Ce script arrête proprement les programmes Go en permettant le traitement complet des messages en cours :
    - Envoie SIGTERM aux processus (arrêt gracieux)
    - Attend jusqu'à 10 secondes pour que tous les messages soient traités
    - Force l'arrêt uniquement si nécessaire après le délai
    - Supprime ensuite les conteneurs Docker

### Manuellement (Sans les Scripts)

Si vous préférez exécuter chaque composant séparément, suivez ces étapes :

1.  **Démarrer l'environnement Docker** :
    ```bash
    docker compose up -d
    ```

2.  **Attendre l'initialisation de Kafka** :
    Après avoir lancé les conteneurs, attendez environ 30 secondes pour que Kafka soit pleinement opérationnel.

3.  **Créer le topic Kafka** (avec plusieurs partitions pour la scalabilité) :
    ```bash
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic orders --partitions 3 --replication-factor 1
    ```
    
    **Note** : Si le topic existe déjà avec 1 partition, vous pouvez augmenter le nombre de partitions :
    ```bash
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --alter --topic orders --partitions 3
    ```

4.  **Lancer le(s) consommateur(s)** :
    
    **Option A : Une seule instance** (pour le développement) :
    ```bash
    go run tracker.go order.go
    ```
    
    **Option B : Plusieurs instances** (pour la scalabilité - pattern Competing Consumers) :
    ```bash
    # Lancer 3 instances dans le même consumer group
    for i in {1..3}; do
        TRACKER_INSTANCE_ID="instance-$i" go run tracker.go order.go &
        sleep 1
    done
    ```
    
    Chaque instance aura ses propres fichiers de logs : `tracker-instance-1.log`, `tracker-instance-2.log`, etc.

5.  **Lancer le producteur** :
    Ouvrez un second terminal et exécutez :
    ```bash
    go run producer.go order.go
    ```

## Observabilité et Logging

Le système de tracking (`tracker.go`) utilise deux fichiers de journalisation distincts pour séparer les préoccupations. **Avec le pattern Competing Consumers, chaque instance a ses propres fichiers de logs** (identifiés par l'ID d'instance).

### Fichiers de Journalisation

**Format des fichiers** : `tracker-<instance-id>.log` et `tracker-<instance-id>.events`

Exemples :
- `tracker-instance-1.log` et `tracker-instance-1.events` (instance 1)
- `tracker-instance-2.log` et `tracker-instance-2.events` (instance 2)
- `tracker-instance-3.log` et `tracker-instance-3.events` (instance 3)

Si aucune variable d'environnement `TRACKER_INSTANCE_ID` n'est définie, l'instance utilisera un ID basé sur le PID : `tracker-instance-<pid>.log`

1. **`tracker-<instance-id>.log`** : **Observabilité système complète**
   - **Événements système** : Démarrage, initialisation, arrêt du consommateur
   - **Métriques périodiques** : Statistiques toutes les 30 secondes (uptime, messages reçus/traités/échoués, taux de succès, débit)
   - **Erreurs** : Erreurs de lecture Kafka, désérialisation, etc.
   - **Statistiques finales** : Métriques complètes à l'arrêt
   - Format structuré JSON pour faciliter le monitoring et l'analyse

2. **`tracker-<instance-id>.events`** : Journalisation complète de tous les messages reçus
   - **Chaque message reçu de Kafka est automatiquement journalisé**
   - Format optimisé pour la traçabilité et l'analyse
   - Inclut le message brut, les métadonnées Kafka, et la structure complète si désérialisée
   - Messages valides ET invalides (avec erreur de désérialisation)

### Garantie de Journalisation

- **Tous les messages reçus sont journalisés dans `tracker.events`** : Chaque message Kafka est enregistré dès sa réception, indépendamment du succès de la désérialisation
- **Messages valides** : Journalisés uniquement dans `tracker.events` avec le message brut ET la structure Order complète
- **Messages invalides** : Journalisés dans `tracker.events` ET dans `tracker.log` (pour le débogage)
- **Observabilité système complète** : `tracker.log` contient :
  - Événements de cycle de vie (démarrage, arrêt)
  - Métriques périodiques (toutes les 30 secondes)
  - Statistiques de performance (uptime, débit, taux de succès)
  - Erreurs système (lecture Kafka, désérialisation, etc.)
- **Aucune perte** : Aucun message n'est perdu, même en cas d'erreur de traitement
- **Séparation des préoccupations** : 
  - `tracker.events` : Journalisation complète de tous les messages (traçabilité)
  - `tracker.log` : Observabilité système complète (métriques, erreurs, événements système)

### Format des Fichiers

#### tracker.log (Observabilité Système)

Format JSON avec les champs suivants selon le type d'événement :

**Événements système (INFO)** :
- `timestamp` : Date et heure de l'événement (RFC3339)
- `level` : `INFO`
- `message` : Description de l'événement
- `service` : Nom du service (order-tracker)
- `metadata` : Métadonnées selon l'événement :
  - **Démarrage** : `log_file`, `events_file`, `start_time`
  - **Initialisation Kafka** : `topic`, `group_id`, `bootstrap_server`, `mode`, `auto_offset_reset`
  - **Métriques périodiques** : `uptime_seconds`, `messages_received`, `messages_processed`, `messages_failed`, `success_rate_percent`, `messages_per_second`, `last_message_time`, `last_processed_offset`
  - **Arrêt** : `signal`, `uptime_seconds`, `total_messages_received`, `total_messages_processed`, `total_messages_failed`, `final_success_rate_percent`, `shutdown_time`

**Erreurs (ERROR)** :
- `timestamp` : Date et heure de l'événement (RFC3339)
- `level` : `ERROR`
- `message` : Description de l'erreur
- `service` : Nom du service (order-tracker)
- `error` : Message d'erreur détaillé
- `metadata` : Métadonnées contextuelles de l'erreur (topic, partition, offset, raw_message, etc.)

#### tracker.events (Journalisation Complète)

Format JSON optimisé pour la traçabilité avec les champs suivants :
- `timestamp` : Date et heure de réception (RFC3339)
- `event_type` : Type d'événement (`message.received` ou `message.received.deserialization_error`)
- `kafka_topic` : Topic Kafka source
- `kafka_partition` : Partition Kafka
- `kafka_offset` : Offset Kafka (pour traçabilité complète)
- `kafka_key` : Clé du message Kafka (si présente)
- `raw_message` : **Le message brut JSON tel que reçu de Kafka** (toujours présent)
- `message_size` : Taille du message en octets
- `deserialized` : Booléen indiquant si la désérialisation a réussi
- `order_id` : ID de la commande (si désérialisée avec succès)
- `sequence` : Numéro de séquence (si désérialisée avec succès)
- `status` : Statut de la commande (si désérialisée avec succès)
- `order_full` : **Structure Order complète en JSON** (si désérialisée avec succès)
- `error` : Message d'erreur (si désérialisation échouée)

### Analyse des Logs et Événements

Un script d'analyse est fourni pour exploiter les logs :

```bash
chmod +x analyze_logs.sh
./analyze_logs.sh
```

Ce script affiche :
- Statistiques générales sur `tracker.log` (événements système, métriques, erreurs)
- Nombre de commandes traitées (depuis `tracker.events`)
- Détection d'erreurs
- Statistiques financières (si `jq` est installé)
- Top clients
- Dernières entrées de log

**Note** : `tracker-<instance-id>.log` contient l'observabilité système (métriques, erreurs, événements). Pour analyser tous les messages, utilisez `tracker-<instance-id>.events`.

**Avec plusieurs instances** : Pour analyser les logs de toutes les instances, vous pouvez utiliser des patterns globaux :
```bash
# Analyser tous les logs de toutes les instances
cat tracker-instance-*.log | jq

# Compter les messages reçus par toutes les instances
cat tracker-instance-*.events | jq -s 'length'

# Agréger les métriques de toutes les instances
grep '"message":"Métriques système"' tracker-instance-*.log | jq
```

### Analyse des Événements (tracker.events)

Le fichier `tracker.events` contient la journalisation complète de tous les messages reçus. Exemples d'analyse :

```bash
# Compter tous les messages reçus
wc -l tracker.events

# Compter les messages désérialisés vs non désérialisés
grep '"deserialized":true' tracker.events | wc -l
grep '"deserialized":false' tracker.events | wc -l

# Extraire tous les messages bruts
jq -r '.raw_message' tracker.events

# Analyser les messages avec erreur de désérialisation
jq 'select(.deserialized == false)' tracker.events

# Reconstruire l'historique complet des commandes
jq 'select(.deserialized == true) | .order_full' tracker.events

# Statistiques par offset Kafka
jq -r '.kafka_offset' tracker.events | sort -n | uniq -c
```

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

# Lister tous les messages bruts reçus (y compris ceux avec erreur de désérialisation)
grep 'Message reçu de Kafka' tracker.log | jq -r '.metadata.raw_message'

# Compter les messages reçus vs les commandes traitées
echo "Messages reçus: $(grep -c 'Message reçu de Kafka' tracker.log)"
echo "Commandes traitées: $(grep -c 'Commande reçue et traitée' tracker.log)"

# Trouver les messages qui ont échoué à la désérialisation
grep 'Erreur lors de la désérialisation' tracker.log | jq

# Analyser les événements dans tracker.events
jq 'select(.deserialized == false)' tracker.events

# Analyser les métriques système dans tracker.log
grep '"message":"Métriques système"' tracker.log | jq

# Suivre l'évolution du taux de succès
grep '"message":"Métriques système"' tracker.log | jq -r '[.timestamp, .metadata.success_rate_percent] | @csv'

# Analyser les statistiques finales à l'arrêt
grep '"message":"Consommateur arrêté proprement"' tracker.log | jq
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

-   **Vérifier la répartition des partitions (Competing Consumers)** :
    Cette commande est essentielle pour vérifier que le pattern Competing Consumers fonctionne correctement. Elle affiche quelle instance consomme quelle partition :
    ```bash
    docker exec kafka kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group order-tracker
    ```
    
    Exemple de sortie :
    ```
    TOPIC     PARTITION  CURRENT-OFFSET  LAG  CONSUMER-ID                    HOST      CLIENT-ID
    orders    0          150             0    consumer-order-tracker-1        /...      tracker-instance-1
    orders    1          145             0    consumer-order-tracker-2        /...      tracker-instance-2
    orders    2          148             0    consumer-order-tracker-3        /...      tracker-instance-3
    ```
    
    Cela confirme que chaque partition est consommée par une instance différente.

-   **Consommer les messages depuis le terminal** :
    Une excellente façon de déboguer ou de visualiser le flux de messages en temps réel.
    ```bash
    docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
    ```

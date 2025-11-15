# Projet de Démonstration Kafka avec Go

Ce projet est une démonstration d'un système de messagerie basé sur Apache Kafka, entièrement conteneurisé avec Docker. Il illustre plusieurs patrons d'architecture orientée événements (EDA) et bonnes pratiques de production à travers un cas d'utilisation simple : un producteur qui génère des commandes enrichies et un consommateur qui les traite de manière autonome et observable.

## Patrons d'Architecture et Bonnes Pratiques

Ce projet met en œuvre plusieurs patrons et pratiques essentiels pour les systèmes distribués.

### 1. Event-Driven Architecture (EDA)
Le système est entièrement piloté par les événements. Le producteur et le consommateur ne communiquent pas directement, mais via des événements (messages de commande) stockés dans Kafka. Cela favorise le découplage, la scalabilité et la résilience.

### 2. Publisher/Subscriber
Le modèle de communication est le Pub/Sub. Le `producer.go` publie des messages dans le topic `orders` sans savoir qui les consommera. Le `tracker.go` s'abonne à ce topic pour recevoir les messages, permettant à plusieurs consommateurs de traiter les mêmes messages en parallèle si nécessaire.

### 3. Event Carried State Transfer
C'est le patron de conception de message le plus important de ce projet. Chaque message de commande est **enrichi avec toutes les données nécessaires à son traitement** (informations client, détails de l'inventaire, etc.). Le consommateur est ainsi **autonome** et n'a pas besoin d'interroger d'autres services, ce qui réduit les dépendances et améliore la latence. Le modèle de données est défini dans `order.go`.

### 4. Audit Trail (Piste d'Audit)
Le fichier `tracker.events` implémente ce patron en créant un **journal immuable de chaque message reçu**. Qu'un message soit valide ou corrompu, il est enregistré. Cette pratique est cruciale pour :
-   L'**audit** : Conserver une preuve de toutes les données entrantes.
-   Le **débogage** : Analyser les messages qui ont causé des erreurs.
-   La **relecture** : Permettre de rejouer des séquences d'événements pour des tests ou une reprise sur erreur.

### 5. Application Health Monitoring
Le fichier `tracker.log` est dédié à la surveillance de la santé de l'application. Il contient des **logs structurés (JSON)** sur les événements de cycle de vie (démarrage, arrêt), les erreurs et les **métriques périodiques** (débit de messages, taux de succès). Ce flux de données est conçu pour alimenter des dashboards, des systèmes d'alerte et des outils d'analyse de logs.

### 6. Guaranteed Delivery (Livraison Garantie)
Le `producer.go` ne se contente pas d'envoyer les messages "à l'aveugle". Il écoute les accusés de réception (delivery reports) de Kafka pour s'assurer que chaque message a bien été reçu et stocké par le broker. La fonction `deliveryReport` dans `producer.go` est responsable de ce suivi.

### 7. Graceful Shutdown (Arrêt Propre)
Le `producer.go` et le `tracker.go` interceptent les signaux du système (comme `Ctrl+C`).
-   Le **producteur** utilise `producer.Flush()` pour envoyer tous les messages qui sont encore dans son tampon.
-   Le **consommateur** termine sa boucle de traitement et ferme proprement sa connexion.
Cela évite la perte de données lors des arrêts planifiés ou des déploiements.

### 8. Gestion Robuste des Processus
Les scripts `start.sh` et `stop.sh` utilisent des fichiers PID (`.pid`) pour une gestion précise des processus. Cela garantit que les signaux d'arrêt sont envoyés aux bons processus, évitant ainsi les arrêts accidentels ou incomplets.

## Stratégie d'Observabilité

Le système utilise une stratégie de journalisation à deux fichiers pour séparer les préoccupations, en s'appuyant sur les patrons décrits ci-dessus :

1.  **`tracker.log` : Journal d'Observabilité (`Application Health Monitoring`)**
    -   **Quoi ?** Événements de cycle de vie, métriques périodiques, et erreurs critiques.
    -   **Pourquoi ?** Pour le **monitoring** et l'**alerte**.

2.  **`tracker.events` : Journal de Traçabilité (`Audit Trail`)**
    -   **Quoi ?** Une copie de **chaque message** reçu de Kafka.
    -   **Pourquoi ?** Pour l'**audit**, le **débogage** et la **relecture**.

## Prérequis

-   **Docker** & **Docker Compose**
-   **Go 1.22+**
-   Optionnel : `jq` pour une analyse avancée des logs JSON.

## 🚀 Démarrage Rapide

1.  **Démarrer l'environnement :**
    ```bash
    ./start.sh
    ```
    Ce script lance Kafka, crée le topic, et exécute le producteur et le consommateur.

2.  **Observer les journaux :**
    Ouvrez deux autres terminaux pour suivre les journaux en temps réel :
    ```bash
    # Suivre les logs système (Application Health Monitoring)
    tail -f tracker.log | jq

    # Suivre tous les messages entrants (Audit Trail)
    tail -f tracker.events | jq
    ```

3.  **Lancer le Moniteur Interactif (Optionnel) :**
    Pour une vue d'ensemble en temps réel, lancez le moniteur de logs :
    ```bash
    go run log_monitor.go
    ```

4.  **Arrêter l'environnement :**
    ```bash
    ./stop.sh
    ```
    Ce script arrête proprement les applications Go, puis les conteneurs Docker.

## Structure du Code

-   **`order.go`** : Définit le modèle de données partagé (Event Carried State Transfer).
-   **`producer.go`** : Le code source du producteur.
-   **`tracker.go`** : Le code source du consommateur.
-   **`log_monitor.go`** : L'interface utilisateur du moniteur de logs.
-   **`docker-compose.yaml`** : Définit le service Kafka.
-   **`start.sh` / `stop.sh`** : Scripts pour gérer le cycle de vie de l'application.

## Commandes Kafka Utiles

-   **Lister les topics :**
    ```bash
    docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
    ```

-   **Consommer les messages depuis le terminal (pour le débogage) :**
    ```bash
    docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
    ```

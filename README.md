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
    go run tracker.go
    ```

5.  **Lancer le producteur** :
    Ouvrez un second terminal et exécutez :
    ```bash
    go run producer.go
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

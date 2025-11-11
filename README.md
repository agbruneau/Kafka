# Projet de Démonstration Kafka avec Golang

Ce projet est une démonstration simple d'un écosystème de messagerie utilisant Apache Kafka. Il comprend un producteur (`producer.go`) qui envoie des messages et un consommateur (`tracker.go`) qui les reçoit. L'ensemble de l'environnement est géré via Docker.

## Prérequis

Avant de commencer, assurez-vous d'avoir installé les outils suivants sur votre machine :

-   **Docker** et **Docker Compose**
-   **Go** (version 1.21 ou supérieure)

## Démarrage de l'Application

Pour lancer l'application, exécutez le script `start.sh` :

```bash
./start.sh
```

Ce script effectue les actions suivantes :
1.  Démarre les conteneurs Docker pour Kafka.
2.  Crée le topic Kafka `orders`.
3.  Compile les programmes Go (`producer.go` et `tracker.go`).
4.  Lance le consommateur (`tracker`) en arrière-plan.
5.  Lance le producteur (`producer`) au premier plan.

## Arrêt de l'Application

Pour arrêter tous les composants de l'application (conteneurs Docker et programmes Go), exécutez le script `stop.sh` :

```bash
./stop.sh
```

## Commandes Utiles pour Kafka

Voici quelques commandes `docker exec` pour interagir directement avec Kafka.

### Lister les Topics

Pour voir la liste de tous les topics Kafka :

```bash
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
```

### Décrire un Topic

Pour obtenir des informations détaillées sur un topic (par exemple, `orders`) :

```bash
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic orders
```

### Écouter les Messages d'un Topic

Pour consommer les messages du topic `orders` directement depuis la ligne de commande et voir ce qui s'y passe en temps réel :

```bash
docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders --from-beginning
```

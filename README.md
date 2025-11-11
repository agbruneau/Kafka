# Projet de Démonstration Kafka avec Python

Ce projet est une démonstration simple d'un écosystème de messagerie utilisant Apache Kafka. Il comprend un producteur (`producer.py`) qui envoie des messages et un consommateur (`tracker.py`) qui les reçoit. L'ensemble de l'environnement est géré via Docker.

## Prérequis

Avant de commencer, assurez-vous d'avoir installé les outils suivants sur votre machine :

-   **Docker** et **Docker Compose**
-   **Python 3**

## Démarrage de l'Application

Pour lancer l'application, exécutez le script `start.sh` :

```bash
./start.sh
```

Ce script effectue les actions suivantes :
1.  Démarre les conteneurs Docker pour Kafka.
2.  Crée le topic Kafka `orders`.
3.  Met en place un environnement virtuel Python et installe les dépendances.
4.  Lance le consommateur (`tracker.py`) en arrière-plan.
5.  Lance le producteur (`producer.py`) au premier plan.

## Arrêt de l'Application

Pour arrêter tous les composants de l'application (conteneurs Docker et scripts Python), exécutez le script `stop.sh` :

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

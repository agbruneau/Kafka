# Projet de Démonstration Kafka avec Go

Ce projet est une démonstration simple d'un écosystème de messagerie utilisant Apache Kafka. Il comprend un producteur (`producer.go`) qui envoie des messages et un consommateur (`tracker.go`) qui les reçoit. L'ensemble de l'environnement est géré via Docker.

## Prérequis

Avant de commencer, assurez-vous d'avoir installé les outils suivants sur votre machine :

-   **Docker** et **Docker Compose**
-   **Go 1.21 ou supérieur**

## Installation des dépendances Go

Pour installer les dépendances Go nécessaires, exécutez :

```bash
go mod download
```

## Compilation des programmes

Pour compiler les programmes Go :

```bash
# Compiler le producteur
go build -o bin/producer producer.go

# Compiler le consommateur
go build -o bin/tracker tracker.go
```

## Démarrage de l'Application

Pour lancer l'application, exécutez le script `start.sh` :

```bash
./start.sh
```

Ce script effectue les actions suivantes :
1.  Démarre les conteneurs Docker pour Kafka.
2.  Crée le topic Kafka `orders`.
3.  Installe les dépendances Go et compile les programmes.
4.  Lance le consommateur (`tracker`) en arrière-plan.
5.  Lance le producteur (`producer`) au premier plan.

## Arrêt de l'Application

Pour arrêter tous les composants de l'application (conteneurs Docker et programmes Go), exécutez le script `stop.sh` :

```bash
./stop.sh
```

## Exécution manuelle

Si vous préférez exécuter les programmes manuellement :

```bash
# Démarrer Kafka
docker-compose up -d

# Créer le topic
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic orders --partitions 1 --replication-factor 1 --if-not-exists

# Lancer le consommateur (dans un terminal)
go run tracker.go

# Lancer le producteur (dans un autre terminal)
go run producer.go
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

## Structure du Projet

- `producer.go` : Programme producteur qui envoie des messages au topic Kafka
- `tracker.go` : Programme consommateur qui lit les messages du topic Kafka
- `go.mod` : Fichier de dépendances Go
- `docker-compose.yaml` : Configuration Docker pour Kafka
- `start.sh` : Script de démarrage
- `stop.sh` : Script d'arrêt

# Pub/Sub événementiel avec Golang et Kafka

Cette démonstration illustre une architecture événementielle complète en Go autour d'Apache Kafka. Elle couvre de nombreux patrons : **Publish/Subscribe**, **Event Streaming**, **Event-Carried State Transfer**, **CQRS**, **Event Sourcing**, **Saga (orchestration + chorégraphie)**, **Choreography**, **Dead Letter Queue**, **Competing Consumers**, **Aggregator**, **Circuit Breaker** et **Service Orchestration**.

## Services Go

| Binaire | Rôle principal |
|---------|----------------|
| `cmd/producer` | Orchestrateur Saga : publie les commandes et événements métier, gère les retries et envoie les messages vers la DLQ en cas d'échec. |
| `cmd/tracker` | Service chorégraphié : consomme `orders.events`, applique des validations idempotentes et publie des compensations si nécessaire. |
| `cmd/aggregator` | Agrégateur d'événements : maintient des agrégats par utilisateur et publie des snapshots sur `orders.aggregates`. |
| `cmd/query-api` | Projection de lecture (CQRS) : restitue les agrégats via HTTP et suit les mises à jour publiées par l’agrégateur. |
| `cmd/dlq-inspector` | Lecture de la dead-letter queue `orders.dlq`. |
| `cmd/replayer` | Relecture Event Sourcing : rejoue `orders.events` pour reconstruire une projection persistée. |

Tous les services réutilisent `internal/bus` (accès Kafka + circuit breaker), `internal/events` (contrats d’événements), `internal/storage` (projections en mémoire) et `internal/metrics` (compteurs + serveur HTTP optionnel).

## Prérequis

- **Docker** et **Docker Compose**
- **Go 1.22 ou supérieur**

## Démarrage

```bash
./start.sh
```

Le script :

1. Démarre Kafka via `docker compose`.
2. Crée les topics nécessaires (`orders.events`, `orders.commands`, `orders.dlq`, `orders.aggregates`).
3. Compile tous les binaires dans `./bin`.
4. Lance en arrière-plan : `tracker`, `aggregator`, `query-api`, `dlq-inspector`.
5. Exécute le producteur orienteur de saga au premier plan.

Les logs sont disponibles dans `./logs` et les PID dans `./pids`.

### Services exposés

- **API de lecture (CQRS)** : http://localhost:8080
  - `GET /aggregates` – liste des agrégats
  - `GET /aggregates/{user}` – détail d’un utilisateur
  - `GET /metrics` – snapshot JSON des compteurs
- **Serveur de métriques** (optionnels) :
  - Agrégateur : `AGGREGATOR_METRICS_PORT=9101`
  - Tracker : `TRACKER_METRICS_PORT=9102`
  - DLQ Inspector : `DLQ_METRICS_PORT=9103` (à définir avant lancement)

Chaque service supporte des workers concurrents via les variables `*_WORKERS`.

## Arrêt

```bash
./stop.sh
```

Ce script tue les processus Go lancés, supprime les binaires générés et arrête les conteneurs Docker.

## Tests

Un test d’intégration (`internal/tests/integration_test.go`) démarre la stack Docker Compose, publie un événement via `internal/bus` et vérifie sa consommation.

```bash
go test ./...
```

> ⚠️ Le test est ignoré si Docker est absent ou si `go test -short` est utilisé.

## Topics Kafka

| Topic | Description |
|-------|-------------|
| `orders.commands` | Intentions (CQRS – écriture), utilisées par l’orchestrateur et les compensations. |
| `orders.events` | Flux principal d’événements métier (Saga, Event Streaming). |
| `orders.aggregates` | Snapshots agrégés pour la lecture (Aggregator + CQRS). |
| `orders.dlq` | Dead Letter Queue pour diagnostics et réinjections manuelles. |

## Gestion des pannes

- **Circuit breaker** (`internal/bus`) limite les appels Kafka lors de pannes.
- **Retries exponentiels** côté producteur avec déroutage vers `orders.dlq`.
- **Compensations** orchestrées/par chorégraphie via `cmd/tracker`.

## Relecture Event Sourcing

```bash
./bin/replayer -output replay-projections.json
```

Le fichier généré contient les agrégats reconstruits depuis `orders.events`.

## Commandes utiles

```bash
docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list
docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders.events --from-beginning
docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic orders.dlq --from-beginning
```

Ces commandes permettent de suivre les flux en parallèle de l’application Go.

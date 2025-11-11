#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$ROOT_DIR/bin"
LOG_DIR="$ROOT_DIR/logs"
PID_DIR="$ROOT_DIR/pids"

mkdir -p "$BIN_DIR" "$LOG_DIR" "$PID_DIR"

echo "🚀 Démarrage de l'environnement Kafka..."
docker compose up -d

echo "⏳ Attente de 30 secondes pour que Kafka démarre complètement..."
sleep 30

create_topic() {
  local topic=$1
  echo "🔥 Vérification du topic '$topic'..."
  docker exec kafka kafka-topics --create --if-not-exists --topic "$topic" \
    --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
}

for topic in orders.events orders.commands orders.dlq orders.aggregates; do
  create_topic "$topic"
done

echo "📦 Installation des dépendances Go..."
go mod download

echo "🔨 Compilation des programmes Go..."
go build -o "$BIN_DIR/producer" ./cmd/producer
go build -o "$BIN_DIR/tracker" ./cmd/tracker
go build -o "$BIN_DIR/aggregator" ./cmd/aggregator
go build -o "$BIN_DIR/query-api" ./cmd/query-api
go build -o "$BIN_DIR/replayer" ./cmd/replayer
go build -o "$BIN_DIR/dlq-inspector" ./cmd/dlq-inspector

start_service() {
  local name=$1
  shift
  local command="$*"

  echo "▶️ Démarrage de $name..."
  bash -c "$command" >"$LOG_DIR/$name.log" 2>&1 &
  local pid=$!
  echo $pid >"$PID_DIR/$name.pid"
}

start_service tracker "TRACKER_METRICS_PORT=9102 \"$BIN_DIR/tracker\""
start_service aggregator "AGGREGATOR_METRICS_PORT=9101 \"$BIN_DIR/aggregator\""
start_service query-api "QUERY_API_PORT=8080 \"$BIN_DIR/query-api\""
start_service dlq-inspector "\"$BIN_DIR/dlq-inspector\""

echo "▶️ Démarrage du producteur (foreground)."
"$BIN_DIR/producer"

echo "✅ Le producteur a terminé. Les services continuent de tourner."
echo "Consultez les logs dans $LOG_DIR et arrêtez l'environnement avec ./stop.sh"

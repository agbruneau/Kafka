#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$ROOT_DIR/bin"
PID_DIR="$ROOT_DIR/pids"

stop_service() {
  local name=$1
  local pid_file="$PID_DIR/$name.pid"
  if [[ -f "$pid_file" ]]; then
    local pid
    pid=$(cat "$pid_file")
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "🛑 Arrêt de $name (PID $pid)..."
      kill "$pid" || true
      wait "$pid" 2>/dev/null || true
    fi
    rm -f "$pid_file"
  fi
}

for service in tracker aggregator query-api dlq-inspector producer; do
  stop_service "$service"
done

echo "🛑 Arrêt des conteneurs Docker..."
docker compose down

echo "🧹 Nettoyage des binaires..."
rm -f "$BIN_DIR"/*

echo "✅ Arrêt complet effectué."

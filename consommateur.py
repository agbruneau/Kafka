"""Exemple de consommateur Kafka en Python.

Ce script illustre la partie "sub" du patron Pub/Sub en se connectant à un topic
Kafka pour lire des messages produits, par exemple, via `producteur.py`.

Avant exécution, assurez-vous que :
  * Kafka est démarré et accessible.
  * Le package `kafka-python` est installé (pip install kafka-python).

Utilisation basique :

    python consommateur.py --topic demo --group-id demo-group
"""
from __future__ import annotations

import argparse
import json
import os
import sys
from dataclasses import dataclass
from typing import Optional

from kafka import KafkaConsumer
from kafka.errors import KafkaError


@dataclass(frozen=True)
class ConsumerConfig:
    """Configuration de base pour le consommateur Kafka."""

    bootstrap_servers: str = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
    group_id: str = os.getenv("KAFKA_GROUP_ID", "demo-group")
    auto_offset_reset: str = os.getenv("KAFKA_AUTO_OFFSET_RESET", "earliest")
    enable_auto_commit: bool = os.getenv("KAFKA_ENABLE_AUTO_COMMIT", "true").lower() == "true"
    session_timeout_ms: int = int(os.getenv("KAFKA_SESSION_TIMEOUT_MS", "10000"))


def build_consumer(config: ConsumerConfig, topic: str) -> KafkaConsumer:
    """Construit un consommateur pointant vers le topic cible."""
    return KafkaConsumer(
        topic,
        bootstrap_servers=config.bootstrap_servers,
        group_id=config.group_id,
        value_deserializer=_safe_json_deserializer,
        key_deserializer=lambda key: key.decode("utf-8") if key is not None else None,
        auto_offset_reset=config.auto_offset_reset,
        enable_auto_commit=config.enable_auto_commit,
        session_timeout_ms=config.session_timeout_ms,
    )


def _safe_json_deserializer(payload: Optional[bytes]) -> Optional[dict]:
    """Désérialise en JSON tout en tolérant des valeurs brutes."""
    if payload is None:
        return None
    try:
        return json.loads(payload.decode("utf-8"))
    except json.JSONDecodeError:
        return {"raw": payload.decode("utf-8", errors="replace")}


def consommer_messages(consumer: KafkaConsumer, limit: Optional[int]) -> None:
    """Lit les messages du topic et les affiche dans la console."""
    print("Attente de messages... (Ctrl+C pour arrêter)")
    compteur = 0
    try:
        for message in consumer:
            compteur += 1
            print(
                f"[REÇU] topic={message.topic} partition={message.partition} "
                f"offset={message.offset} clé={message.key} valeur={message.value}"
            )
            if limit is not None and compteur >= limit:
                print(f"Limite de {limit} messages atteinte, arrêt du consommateur.")
                break
    except KeyboardInterrupt:
        print("Interruption utilisateur. Fermeture du consommateur...")
    except KafkaError as exc:  # pragma: no cover - sortie directe en cas d'erreur
        print(f"[ERREUR] La consommation a échoué: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc
    finally:
        consumer.close()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Consommateur Kafka simple en Python.")
    parser.add_argument("--topic", required=True, help="Nom du topic Kafka à écouter.")
    parser.add_argument(
        "--group-id",
        default=None,
        help="Identifiant du groupe de consommation (sinon valeur d'environnement ou défaut).",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Nombre maximum de messages à lire avant arrêt.",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    config = ConsumerConfig()
    if args.group_id is not None:
        config = ConsumerConfig(
            bootstrap_servers=config.bootstrap_servers,
            group_id=args.group_id,
            auto_offset_reset=config.auto_offset_reset,
            enable_auto_commit=config.enable_auto_commit,
            session_timeout_ms=config.session_timeout_ms,
        )

    consumer = build_consumer(config, args.topic)

    print(
        f"Abonné au topic `{args.topic}` via {config.bootstrap_servers} "
        f"(group_id={config.group_id}, auto_offset_reset={config.auto_offset_reset})"
    )
    consommer_messages(consumer, args.limit)
    print("Consommation terminée.")


if __name__ == "__main__":
    main()

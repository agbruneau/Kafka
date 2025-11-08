"""Exemple de producteur Kafka en Python.

Ce script illustre comment publier des messages dans un topic Kafka
au sein d'un scénario Pub/Sub. Il s'appuie sur la bibliothèque
`kafka-python` (https://kafka-python.readthedocs.io/).

Utilisation basique :

    python producteur.py --topic demo --message "Bonjour Kafka"

Vous pouvez fournir plusieurs occurrences de `--message`. À défaut, le script
enverra une séquence de messages JSON générés automatiquement.
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
from dataclasses import dataclass
from typing import Iterable, Optional

from kafka import KafkaProducer
from kafka.errors import KafkaError


@dataclass(frozen=True)
class ProducerConfig:
    """Configuration de base pour le producteur Kafka."""

    bootstrap_servers: str = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
    acks: str = os.getenv("KAFKA_ACKS", "all")
    linger_ms: int = int(os.getenv("KAFKA_LINGER_MS", "5"))
    retries: int = int(os.getenv("KAFKA_RETRIES", "3"))


def build_producer(config: ProducerConfig) -> KafkaProducer:
    """Initialise et renvoie un producteur Kafka avec une sérialisation JSON."""
    return KafkaProducer(
        bootstrap_servers=config.bootstrap_servers,
        value_serializer=lambda value: json.dumps(value, ensure_ascii=False).encode("utf-8"),
        key_serializer=lambda key: key.encode("utf-8") if key is not None else None,
        linger_ms=config.linger_ms,
        acks=config.acks,
        retries=config.retries,
    )


def iter_payloads(messages: list[str] | None) -> Iterable[dict[str, str]]:
    """Prépare un flux de payloads JSON à partir d'entrées utilisateur ou d'un exemple."""
    if messages:
        for idx, message in enumerate(messages, start=1):
            yield {"id": idx, "payload": message, "source": "cli"}
    else:
        timestamp = int(time.time())
        for idx in range(1, 6):
            yield {
                "id": idx,
                "payload": f"message automatique #{idx}",
                "source": "auto",
                "timestamp": timestamp + idx,
            }


def envoyer_messages(
    producer: KafkaProducer,
    topic: str,
    payloads: Iterable[dict[str, str]],
    key: Optional[str],
) -> None:
    """Envoie un lot de payloads sur le topic indiqué."""
    for payload in payloads:
        future = producer.send(topic, value=payload, key=key)
        try:
            metadata = future.get(timeout=10)
            print(
                f"[OK] Topic={metadata.topic} partition={metadata.partition} "
                f"offset={metadata.offset} message={payload}"
            )
        except KafkaError as exc:  # pragma: no cover - sortie directe en cas d'erreur
            print(f"[ERREUR] L'envoi a échoué: {exc}", file=sys.stderr)
            producer.flush()
            raise SystemExit(1) from exc

    producer.flush()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Producteur Kafka simple en Python.")
    parser.add_argument("--topic", required=True, help="Nom du topic Kafka cible.")
    parser.add_argument(
        "--message",
        action="append",
        dest="messages",
        help="Message texte à envoyer (option répétable).",
    )
    parser.add_argument(
        "--key",
        default=None,
        help="Clé de partition facultative (chaîne).",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    config = ProducerConfig()
    producer = build_producer(config)

    print(
        f"Publication vers `{args.topic}` via {config.bootstrap_servers} "
        f"(acks={config.acks}, linger_ms={config.linger_ms}, retries={config.retries})"
    )
    payloads = iter_payloads(args.messages)
    envoyer_messages(producer, args.topic, payloads, args.key)
    print("Terminé.")


if __name__ == "__main__":
    main()

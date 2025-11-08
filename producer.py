"""
Ce script Python, `producer.py`, est conçu pour fonctionner comme un producteur de messages pour Apache Kafka.
Il envoie des messages JSON sérialisés à un topic Kafka spécifié.

Le script est configuré pour se connecter à un serveur Kafka fonctionnant sur `localhost:9092`.
Il envoie un message prédéfini au topic `orders` et attend une confirmation de livraison.

Fonctionnalités:
- Configuration et initialisation d'un producteur Kafka.
- Envoi d'un message unique au format JSON.
- Rapport de livraison pour confirmer que le message a été bien reçu par le broker Kafka.
"""

import json
import uuid
from confluent_kafka import Producer

def delivery_report(err, msg):
    """
    Rapporte le résultat de la livraison d'un message.

    Cette fonction de rappel est déclenchée une fois que le message a été livré
    au broker Kafka ou si une erreur est survenue.

    Args:
        err (KafkaError): Une erreur si la livraison a échoué.
        msg (Message): Le message qui a été livré.
    """
    if err:
        print(f"❌ La livraison a échoué: {err}")
    else:
        print(f"✅ Message livré à {msg.topic()} [{msg.partition()}] @ offset {msg.offset()}")
        print(f"   Contenu: {msg.value().decode('utf-8')}")

def main():
    """
    Point d'entrée principal du script producteur.

    Initialise le producteur Kafka, envoie un message de commande au topic 'orders'
    et attend la confirmation de livraison avant de terminer.
    """
    producer_config = {
        "bootstrap.servers": "localhost:9092"
    }

    producer = Producer(producer_config)

    order = {
        "order_id": str(uuid.uuid4()),
        "user": "lara",
        "item": "frozen yogurt",
        "quantity": 10
    }

    value = json.dumps(order).encode("utf-8")

    try:
        producer.produce(
            topic="orders",
            value=value,
            callback=delivery_report
        )
        producer.flush()
    except BufferError:
        print("La file d'attente locale du producteur est pleine, attente...")
        producer.flush()

if __name__ == '__main__':
    main()
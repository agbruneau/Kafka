"""
Ce script Python, `producer.py`, est conçu pour fonctionner comme un producteur de messages pour Apache Kafka.
Il envoie des messages JSON sérialisés à un topic Kafka spécifié.

Le script est configuré pour se connecter à un serveur Kafka fonctionnant sur `localhost:9092`.
Il envoie en continu des messages prédéfinis au topic `orders` et attend une confirmation de livraison.

Fonctionnalités:
- Configuration et initialisation d'un producteur Kafka.
- Envoi de messages en continu au format JSON.
- Rapport de livraison pour confirmer que les messages ont été bien reçus par le broker Kafka.
"""

import json
import time
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

    Initialise le producteur Kafka, envoie des messages de commande en boucle
    au topic 'orders' et attend la confirmation de livraison avant de terminer.
    """
    producer_config = {
        "bootstrap.servers": "localhost:9092"
    }

    producer = Producer(producer_config)

    try:
        while True:
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
                # Attendre que les messages soient envoyés et les callbacks traités
                producer.poll(0)
                producer.flush(1)  # Forcer l'envoi du message
            except BufferError:
                print("La file d'attente locale du producteur est pleine, attente...")
                producer.flush()

            # Attendre 2 secondes avant d'envoyer le prochain message
            time.sleep(2)

    except KeyboardInterrupt:
        print("\n🔴 Arrêt du producteur")
    finally:
        # S'assurer que tous les messages restants sont envoyés avant de fermer
        print("⏳ Envoi des messages restants...")
        producer.flush()

if __name__ == '__main__':
    main()

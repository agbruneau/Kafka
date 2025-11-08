"""
Ce script Python, `tracker.py`, est un consommateur de messages pour Apache Kafka.
Il est conçu pour suivre les messages d'un topic Kafka spécifié, les désérialiser
et afficher les informations qu'ils contiennent.

Le script est configuré pour se connecter à un serveur Kafka fonctionnant sur `localhost:9092`
et s'abonner au topic `orders`. Il écoute en continu les nouveaux messages et les
affiche dans la console.

Fonctionnalités:
- Configuration et initialisation d'un consommateur Kafka.
- Abonnement à un topic Kafka.
- Boucle de consommation pour recevoir et traiter les messages en temps réel.
- Gestion des erreurs et fermeture propre du consommateur.
"""

import json
from confluent_kafka import Consumer

def main():
    """
    Point d'entrée principal du script consommateur.

    Initialise un consommateur Kafka, s'abonne au topic 'orders' et entre dans une
    boucle infinie pour recevoir et traiter les messages.
    """
    consumer_config = {
        "bootstrap.servers": "localhost:9092",
        "group.id": "order-tracker",
        "auto.offset.reset": "earliest"
    }

    consumer = Consumer(consumer_config)
    consumer.subscribe(["orders"])

    print("🟢 Le consommateur est en cours d'exécution et abonné au topic 'orders'")

    try:
        while True:
            msg = consumer.poll(1.0)
            if msg is None:
                continue
            if msg.error():
                print("❌ Erreur:", msg.error())
                continue

            value = msg.value().decode("utf-8")
            order = json.loads(value)
            print(f"📦 Commande reçue: {order['quantity']} x {order['item']} de {order['user']}")

    except KeyboardInterrupt:
        print("\n🔴 Arrêt du consommateur")

    finally:
        consumer.close()

if __name__ == '__main__':
    main()
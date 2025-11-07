import json

from confluent_kafka import Consumer

def main():
    """Initialise un consommateur Kafka, s'abonne à un topic et traite les messages.

    Le consommateur est configuré pour se connecter à un broker Kafka sur `localhost:9092`
    et s'abonne au topic `orders`. Il écoute en continu les nouveaux messages et
    affiche les informations sur les commandes reçues. La boucle du consommateur peut être
    arrêtée avec une interruption clavier (Ctrl+C), ce qui garantit une
    fermeture propre du consommateur.
    """
    consumer_config = {
        "bootstrap.servers": "localhost:9092",
        "group.id": "order-tracker",
        "auto.offset.reset": "earliest"
    }

    consumer = Consumer(consumer_config)
    consumer.subscribe(["orders"])
    print("🟢 Consumer is running and subscribed to orders topic")

    try:
        while True:
            msg = consumer.poll(1.0)
            if msg is None:
                continue
            if msg.error():
                print("❌ Error:", msg.error())
                continue

            value = msg.value().decode("utf-8")
            order = json.loads(value)
            print(f"📦 Received order: {order['quantity']} x {order['item']} from {order['user']}")
    except KeyboardInterrupt:
        print("\n🔴 Stopping consumer")
    finally:
        consumer.close()

if __name__ == "__main__":
    main()
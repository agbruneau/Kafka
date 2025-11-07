import json
import uuid

from confluent_kafka import Producer

def main():
    """Crée un producteur Kafka, génère un message de commande et l'envoie à un topic Kafka.

    Le producteur est configuré pour se connecter à un broker Kafka sur `localhost:9092`.
    Le message de commande contient un ID de commande, un nom d'utilisateur, un article et une quantité.
    Le message est ensuite sérialisé en JSON et envoyé au topic `orders`.
    La fonction attend également la fin de l'envoi du message avant de se terminer.
    """
    producer_config = {
        "bootstrap.servers": "localhost:9092"
    }

    producer = Producer(producer_config)

    def delivery_report(err, msg):
        """Rapporte le résultat de la livraison d'un message.

        Cette fonction de rappel est déclenchée une fois que le message est livré au broker
        ou si une erreur se produit. Elle affiche un message de succès ou d'échec
        en fonction du résultat.

        Args:
            err (KafkaError): Une erreur si la livraison a échoué, sinon None.
            msg (Message): Le message qui a été livré.
        """
        if err:
            print(f"❌ Delivery failed: {err}")
        else:
            print(f"✅ Delivered {msg.value().decode('utf-8')}")
            print(f"✅ Delivered to {msg.topic()} : partition {msg.partition()} : at offset {msg.offset()}")

    order = {
        "order_id": str(uuid.uuid4()),
        "user": "lara",
        "item": "frozen yogurt",
        "quantity": 10
    }

    value = json.dumps(order).encode("utf-8")

    producer.produce(
        topic="orders",
        value=value,
        callback=delivery_report
    )

    producer.flush()

if __name__ == "__main__":
    main()
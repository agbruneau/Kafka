"""
Producteur Kafka - Exemple éducatif du patron Pub-Sub
Ce script envoie des messages à un topic Kafka
"""

from kafka import KafkaProducer
from kafka.errors import KafkaError
import json
import time
import logging
from datetime import datetime

# Configuration du logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class ProducteurKafka:
    """Classe pour gérer la production de messages vers Kafka"""
    
    def __init__(self, bootstrap_servers='localhost:9092', topic='demo-topic'):
        """
        Initialise le producteur Kafka
        
        Args:
            bootstrap_servers (str): Adresse du serveur Kafka
            topic (str): Nom du topic Kafka
        """
        self.topic = topic
        self.bootstrap_servers = bootstrap_servers
        
        try:
            # Création du producteur avec sérialisation JSON
            self.producer = KafkaProducer(
                bootstrap_servers=self.bootstrap_servers,
                value_serializer=lambda v: json.dumps(v).encode('utf-8'),
                key_serializer=lambda k: k.encode('utf-8') if k else None,
                acks='all',  # Attendre la confirmation de tous les réplicas
                retries=3,   # Nombre de tentatives en cas d'échec
                max_in_flight_requests_per_connection=1  # Garantir l'ordre des messages
            )
            logger.info(f"Producteur Kafka initialisé avec succès sur {bootstrap_servers}")
        except Exception as e:
            logger.error(f"Erreur lors de l'initialisation du producteur: {e}")
            raise
    
    def envoyer_message(self, message, cle=None):
        """
        Envoie un message au topic Kafka
        
        Args:
            message (dict): Message à envoyer (sera converti en JSON)
            cle (str, optional): Clé du message pour le partitionnement
            
        Returns:
            bool: True si l'envoi a réussi, False sinon
        """
        try:
            # Envoi asynchrone du message
            future = self.producer.send(
                self.topic,
                key=cle,
                value=message
            )
            
            # Attendre la confirmation (synchrone pour cet exemple)
            record_metadata = future.get(timeout=10)
            
            logger.info(
                f"Message envoyé avec succès - Topic: {record_metadata.topic}, "
                f"Partition: {record_metadata.partition}, "
                f"Offset: {record_metadata.offset}"
            )
            return True
            
        except KafkaError as e:
            logger.error(f"Erreur Kafka lors de l'envoi du message: {e}")
            return False
        except Exception as e:
            logger.error(f"Erreur inattendue lors de l'envoi du message: {e}")
            return False
    
    def fermer(self):
        """Ferme proprement le producteur"""
        try:
            self.producer.flush()  # Assurer que tous les messages sont envoyés
            self.producer.close()
            logger.info("Producteur Kafka fermé proprement")
        except Exception as e:
            logger.error(f"Erreur lors de la fermeture du producteur: {e}")


def demo_producteur():
    """Fonction de démonstration du producteur"""
    
    # Configuration
    KAFKA_SERVER = 'localhost:9092'
    TOPIC = 'demo-topic'
    
    # Création du producteur
    producteur = ProducteurKafka(
        bootstrap_servers=KAFKA_SERVER,
        topic=TOPIC
    )
    
    try:
        logger.info("Début de l'envoi de messages de démonstration...")
        
        # Envoi de 10 messages de test
        for i in range(10):
            message = {
                'id': i,
                'timestamp': datetime.now().isoformat(),
                'type': 'demo',
                'contenu': f'Ceci est un message de test numéro {i}',
                'donnees': {
                    'temperature': 20 + i,
                    'humidite': 50 + (i * 2),
                    'statut': 'actif'
                }
            }
            
            # Utiliser l'id comme clé pour garantir l'ordre des messages avec le même id
            cle = f"message-{i % 3}"  # 3 clés différentes pour montrer le partitionnement
            
            succes = producteur.envoyer_message(message, cle=cle)
            
            if succes:
                logger.info(f"Message {i} envoyé: {message['contenu']}")
            else:
                logger.error(f"Échec de l'envoi du message {i}")
            
            # Pause pour simuler un flux de données réaliste
            time.sleep(1)
        
        logger.info("Tous les messages ont été traités")
        
    except KeyboardInterrupt:
        logger.info("Interruption par l'utilisateur")
    except Exception as e:
        logger.error(f"Erreur dans la démonstration: {e}")
    finally:
        # Fermeture propre du producteur
        producteur.fermer()


if __name__ == "__main__":
    print("=" * 60)
    print("PRODUCTEUR KAFKA - Démonstration du patron Pub-Sub")
    print("=" * 60)
    print("\nAssurez-vous que Kafka est démarré sur localhost:9092")
    print("Topic utilisé: demo-topic\n")
    
    demo_producteur()

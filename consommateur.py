"""
Consommateur Kafka - Exemple éducatif du patron Pub-Sub
Ce script consomme des messages depuis un topic Kafka
"""

from kafka import KafkaConsumer
from kafka.errors import KafkaError
import json
import logging
import signal
import sys

# Configuration du logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class ConsommateurKafka:
    """Classe pour gérer la consommation de messages depuis Kafka"""
    
    def __init__(self, bootstrap_servers='localhost:9092', topic='demo-topic', 
                 group_id='demo-group'):
        """
        Initialise le consommateur Kafka
        
        Args:
            bootstrap_servers (str): Adresse du serveur Kafka
            topic (str): Nom du topic Kafka
            group_id (str): Identifiant du groupe de consommateurs
        """
        self.topic = topic
        self.bootstrap_servers = bootstrap_servers
        self.group_id = group_id
        self.running = True
        
        try:
            # Création du consommateur avec désérialisation JSON
            self.consumer = KafkaConsumer(
                self.topic,
                bootstrap_servers=self.bootstrap_servers,
                group_id=self.group_id,
                value_deserializer=lambda m: json.loads(m.decode('utf-8')),
                key_deserializer=lambda k: k.decode('utf-8') if k else None,
                auto_offset_reset='earliest',  # Lire depuis le début si pas d'offset sauvegardé
                enable_auto_commit=True,       # Commit automatique des offsets
                auto_commit_interval_ms=1000,  # Interval de commit automatique
                max_poll_records=10            # Nombre max de messages par poll
            )
            logger.info(
                f"Consommateur Kafka initialisé avec succès - "
                f"Server: {bootstrap_servers}, Topic: {topic}, Group: {group_id}"
            )
        except Exception as e:
            logger.error(f"Erreur lors de l'initialisation du consommateur: {e}")
            raise
    
    def traiter_message(self, message):
        """
        Traite un message reçu
        
        Args:
            message: Message Kafka reçu
        """
        try:
            logger.info(
                f"\n{'='*60}\n"
                f"MESSAGE REÇU:\n"
                f"  Topic: {message.topic}\n"
                f"  Partition: {message.partition}\n"
                f"  Offset: {message.offset}\n"
                f"  Clé: {message.key}\n"
                f"  Timestamp: {message.timestamp}\n"
                f"  Contenu: {json.dumps(message.value, indent=2, ensure_ascii=False)}\n"
                f"{'='*60}"
            )
            
            # Vous pouvez ajouter ici votre logique métier personnalisée
            # Par exemple, sauvegarder dans une base de données, envoyer une notification, etc.
            
            return True
            
        except Exception as e:
            logger.error(f"Erreur lors du traitement du message: {e}")
            return False
    
    def consommer_messages(self):
        """Boucle principale de consommation des messages"""
        try:
            logger.info("Début de la consommation des messages...")
            logger.info("Appuyez sur Ctrl+C pour arrêter\n")
            
            # Boucle de consommation
            for message in self.consumer:
                if not self.running:
                    break
                
                # Traiter chaque message
                self.traiter_message(message)
                
        except KeyboardInterrupt:
            logger.info("\nInterruption par l'utilisateur")
        except KafkaError as e:
            logger.error(f"Erreur Kafka: {e}")
        except Exception as e:
            logger.error(f"Erreur inattendue: {e}")
        finally:
            self.fermer()
    
    def arreter(self):
        """Arrête la consommation de messages"""
        logger.info("Arrêt du consommateur en cours...")
        self.running = False
    
    def fermer(self):
        """Ferme proprement le consommateur"""
        try:
            self.consumer.close()
            logger.info("Consommateur Kafka fermé proprement")
        except Exception as e:
            logger.error(f"Erreur lors de la fermeture du consommateur: {e}")


def signal_handler(sig, frame, consommateur):
    """Gestionnaire de signal pour arrêt propre"""
    logger.info('\nSignal reçu, arrêt en cours...')
    consommateur.arreter()
    sys.exit(0)


def demo_consommateur():
    """Fonction de démonstration du consommateur"""
    
    # Configuration
    KAFKA_SERVER = 'localhost:9092'
    TOPIC = 'demo-topic'
    GROUP_ID = 'demo-group'
    
    # Création du consommateur
    consommateur = ConsommateurKafka(
        bootstrap_servers=KAFKA_SERVER,
        topic=TOPIC,
        group_id=GROUP_ID
    )
    
    # Configuration du gestionnaire de signal pour Ctrl+C
    signal.signal(
        signal.SIGINT, 
        lambda sig, frame: signal_handler(sig, frame, consommateur)
    )
    
    # Démarrage de la consommation
    consommateur.consommer_messages()


def demo_consommateur_multiple():
    """
    Démonstration avec plusieurs consommateurs dans le même groupe
    (à exécuter dans des terminaux séparés pour voir le partitionnement)
    """
    
    print("\n" + "="*60)
    print("CONSOMMATEUR MULTIPLE - Démonstration du Load Balancing")
    print("="*60)
    print("\nPour voir le partitionnement:")
    print("1. Lancez ce script dans plusieurs terminaux")
    print("2. Tous les consommateurs avec le même group_id se partageront les messages")
    print("3. Chaque message sera traité par un seul consommateur du groupe\n")
    
    demo_consommateur()


if __name__ == "__main__":
    print("=" * 60)
    print("CONSOMMATEUR KAFKA - Démonstration du patron Pub-Sub")
    print("=" * 60)
    print("\nAssurez-vous que:")
    print("  1. Kafka est démarré sur localhost:9092")
    print("  2. Le topic 'demo-topic' existe (sera créé automatiquement)")
    print("  3. Le producteur envoie des messages\n")
    
    print("Options:")
    print("  1. Consommateur simple (défaut)")
    print("  2. Démonstration consommateur multiple\n")
    
    choix = input("Votre choix (1 ou 2, Entrée pour 1): ").strip() or "1"
    
    if choix == "2":
        demo_consommateur_multiple()
    else:
        demo_consommateur()

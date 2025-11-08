#!/usr/bin/env python3
"""
Producteur Kafka - Exemple éducatif de Pub-Sub
Ce script envoie des messages à un topic Kafka.
"""

import json
import time
from datetime import datetime
from kafka import KafkaProducer
from kafka.errors import KafkaError


class ProducteurKafka:
    """Classe pour produire des messages vers Kafka"""
    
    def __init__(self, bootstrap_servers='localhost:9092', topic='mon-topic'):
        """
        Initialise le producteur Kafka
        
        Args:
            bootstrap_servers: Adresse du serveur Kafka (par défaut: localhost:9092)
            topic: Nom du topic où envoyer les messages (par défaut: mon-topic)
        """
        self.topic = topic
        self.producer = KafkaProducer(
            bootstrap_servers=bootstrap_servers,
            value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            key_serializer=lambda k: k.encode('utf-8') if k else None,
            # Options de fiabilité
            acks='all',  # Attendre confirmation de tous les réplicas
            retries=3,   # Nombre de tentatives en cas d'échec
        )
        print(f"✅ Producteur Kafka initialisé pour le topic: {self.topic}")
    
    def envoyer_message(self, message, key=None):
        """
        Envoie un message au topic Kafka
        
        Args:
            message: Dictionnaire contenant le message à envoyer
            key: Clé optionnelle pour partitionner les messages
        """
        try:
            future = self.producer.send(self.topic, value=message, key=key)
            # Attendre la confirmation que le message a été envoyé
            record_metadata = future.get(timeout=10)
            print(f"✅ Message envoyé avec succès!")
            print(f"   Topic: {record_metadata.topic}")
            print(f"   Partition: {record_metadata.partition}")
            print(f"   Offset: {record_metadata.offset}")
            return record_metadata
        except KafkaError as e:
            print(f"❌ Erreur lors de l'envoi du message: {e}")
            return None
    
    def fermer(self):
        """Ferme la connexion du producteur"""
        self.producer.close()
        print("🔒 Producteur fermé")


def main():
    """Fonction principale pour démonstration"""
    # Configuration
    BOOTSTRAP_SERVERS = 'localhost:9092'
    TOPIC = 'mon-topic'
    
    # Créer le producteur
    producteur = ProducteurKafka(
        bootstrap_servers=BOOTSTRAP_SERVERS,
        topic=TOPIC
    )
    
    try:
        # Envoyer quelques messages de démonstration
        print("\n📤 Envoi de messages...\n")
        
        for i in range(5):
            message = {
                'id': i + 1,
                'timestamp': datetime.now().isoformat(),
                'contenu': f'Message numéro {i + 1}',
                'type': 'demo',
                'auteur': 'producteur-python'
            }
            
            # Utiliser l'ID comme clé pour garantir l'ordre dans la même partition
            producteur.envoyer_message(message, key=f'msg-{i + 1}')
            print()  # Ligne vide pour la lisibilité
            
            # Attendre un peu entre les messages
            time.sleep(1)
        
        print("✨ Tous les messages ont été envoyés avec succès!")
        
    except KeyboardInterrupt:
        print("\n⚠️  Interruption par l'utilisateur")
    except Exception as e:
        print(f"\n❌ Erreur: {e}")
    finally:
        producteur.fermer()


if __name__ == '__main__':
    main()

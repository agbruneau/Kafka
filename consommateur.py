#!/usr/bin/env python3
"""
Consommateur Kafka - Exemple éducatif de Pub-Sub
Ce script consomme des messages depuis un topic Kafka.
"""

import json
from datetime import datetime
from kafka import KafkaConsumer
from kafka.errors import KafkaError


class ConsommateurKafka:
    """Classe pour consommer des messages depuis Kafka"""
    
    def __init__(self, bootstrap_servers='localhost:9092', topic='mon-topic', 
                 group_id='mon-groupe-consommateur', auto_offset_reset='earliest'):
        """
        Initialise le consommateur Kafka
        
        Args:
            bootstrap_servers: Adresse du serveur Kafka (par défaut: localhost:9092)
            topic: Nom du topic à consommer (par défaut: mon-topic)
            group_id: ID du groupe de consommateurs (par défaut: mon-groupe-consommateur)
            auto_offset_reset: Où commencer à lire ('earliest' ou 'latest')
        """
        self.topic = topic
        self.consumer = KafkaConsumer(
            topic,
            bootstrap_servers=bootstrap_servers,
            group_id=group_id,
            auto_offset_reset=auto_offset_reset,
            enable_auto_commit=True,  # Commit automatique des offsets
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            key_deserializer=lambda k: k.decode('utf-8') if k else None,
            # Options de consommation
            consumer_timeout_ms=1000,  # Timeout pour éviter de bloquer indéfiniment
        )
        print(f"✅ Consommateur Kafka initialisé")
        print(f"   Topic: {self.topic}")
        print(f"   Groupe: {group_id}")
        print(f"   Offset initial: {auto_offset_reset}")
    
    def consommer_messages(self, nombre_messages=None):
        """
        Consomme des messages depuis le topic
        
        Args:
            nombre_messages: Nombre de messages à consommer (None = infini)
        """
        print(f"\n📥 Début de la consommation des messages...\n")
        
        messages_consommes = 0
        
        try:
            for message in self.consumer:
                # Extraire les informations du message
                topic = message.topic
                partition = message.partition
                offset = message.offset
                key = message.key
                value = message.value
                
                # Afficher le message reçu
                print("=" * 60)
                print(f"📨 Message reçu:")
                print(f"   Topic: {topic}")
                print(f"   Partition: {partition}")
                print(f"   Offset: {offset}")
                if key:
                    print(f"   Clé: {key}")
                print(f"   Contenu: {json.dumps(value, indent=2, ensure_ascii=False)}")
                print("=" * 60)
                print()
                
                messages_consommes += 1
                
                # Arrêter après avoir consommé le nombre demandé de messages
                if nombre_messages and messages_consommes >= nombre_messages:
                    print(f"✅ {messages_consommes} message(s) consommé(s)")
                    break
                    
        except KafkaError as e:
            print(f"❌ Erreur Kafka: {e}")
        except KeyboardInterrupt:
            print(f"\n⚠️  Interruption par l'utilisateur")
            print(f"✅ {messages_consommes} message(s) consommé(s) avant l'interruption")
        except Exception as e:
            print(f"❌ Erreur inattendue: {e}")
    
    def consommer_en_continu(self):
        """Consomme les messages en continu jusqu'à interruption"""
        print(f"\n📥 Consommation en continu (Ctrl+C pour arrêter)...\n")
        
        try:
            for message in self.consumer:
                topic = message.topic
                partition = message.partition
                offset = message.offset
                key = message.key
                value = message.value
                
                print("=" * 60)
                print(f"📨 [{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}] Message reçu:")
                print(f"   Topic: {topic} | Partition: {partition} | Offset: {offset}")
                if key:
                    print(f"   Clé: {key}")
                print(f"   Contenu: {json.dumps(value, indent=2, ensure_ascii=False)}")
                print("=" * 60)
                print()
                
        except KeyboardInterrupt:
            print(f"\n⚠️  Arrêt du consommateur")
        except Exception as e:
            print(f"❌ Erreur: {e}")
    
    def fermer(self):
        """Ferme la connexion du consommateur"""
        self.consumer.close()
        print("🔒 Consommateur fermé")


def main():
    """Fonction principale pour démonstration"""
    # Configuration
    BOOTSTRAP_SERVERS = 'localhost:9092'
    TOPIC = 'mon-topic'
    GROUP_ID = 'mon-groupe-consommateur'
    
    # Créer le consommateur
    consommateur = ConsommateurKafka(
        bootstrap_servers=BOOTSTRAP_SERVERS,
        topic=TOPIC,
        group_id=GROUP_ID,
        auto_offset_reset='earliest'  # Lire depuis le début du topic
    )
    
    try:
        # Consommer les messages (en continu)
        consommateur.consommer_en_continu()
    except Exception as e:
        print(f"❌ Erreur: {e}")
    finally:
        consommateur.fermer()


if __name__ == '__main__':
    main()

# Système Pub-Sub avec Kafka et Python

Exemple éducatif implémentant le patron d'architecture **Publish-Subscribe (Pub-Sub)** utilisant Apache Kafka et Python.

## 📋 Description

Ce projet démontre le fonctionnement du patron Pub-Sub avec :
- **`producteur.py`** : Publie des messages vers un topic Kafka
- **`consommateur.py`** : Souscrit et consomme des messages depuis un topic Kafka

## 🎯 Objectifs pédagogiques

- Comprendre le patron d'architecture Pub-Sub
- Découvrir Apache Kafka comme système de messagerie distribué
- Apprendre à créer des producteurs et consommateurs Kafka en Python
- Observer le partitionnement et le load balancing des messages

## 📦 Prérequis

### 1. Apache Kafka

Vous devez avoir Apache Kafka installé et démarré. 

#### Installation rapide avec Docker :

```bash
# Télécharger et démarrer Kafka avec Docker Compose
docker-compose up -d
```

#### Ou avec Docker directement :

```bash
# 1. Démarrer Zookeeper
docker run -d --name zookeeper -p 2181:2181 zookeeper:latest

# 2. Démarrer Kafka
docker run -d --name kafka -p 9092:9092 \
  --link zookeeper:zookeeper \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  confluentinc/cp-kafka:latest
```

### 2. Python et dépendances

```bash
# Python 3.7 ou supérieur
python3 --version

# Installer les dépendances
pip install -r requirements.txt
```

## 🚀 Utilisation

### Étape 1 : Démarrer le consommateur

Dans un terminal, lancez le consommateur (il va attendre les messages) :

```bash
python3 consommateur.py
```

Le consommateur affichera :
- Les métadonnées de chaque message reçu (topic, partition, offset)
- Le contenu complet du message au format JSON

### Étape 2 : Démarrer le producteur

Dans un **autre terminal**, lancez le producteur :

```bash
python3 producteur.py
```

Le producteur va :
- Envoyer 10 messages de démonstration
- Afficher la confirmation d'envoi pour chaque message
- Utiliser différentes clés pour montrer le partitionnement

### Observation

Vous devriez voir :
1. Le producteur envoie des messages toutes les secondes
2. Le consommateur reçoit et affiche ces messages en temps réel
3. Les métadonnées montrent comment Kafka gère les messages (partitions, offsets)

## 🔧 Configuration

### Modifier le serveur Kafka

Dans les deux fichiers, changez la variable `KAFKA_SERVER` :

```python
KAFKA_SERVER = 'localhost:9092'  # Votre serveur Kafka
```

### Modifier le topic

```python
TOPIC = 'demo-topic'  # Nom de votre topic
```

### Modifier le groupe de consommateurs

Dans `consommateur.py` :

```python
GROUP_ID = 'demo-group'  # Identifiant du groupe
```

## 🎓 Concepts clés

### Patron Pub-Sub

- **Découplage** : Le producteur et le consommateur ne se connaissent pas
- **Asynchrone** : Les messages sont traités de manière asynchrone
- **Scalabilité** : Plusieurs consommateurs peuvent traiter les messages en parallèle

### Kafka

- **Topic** : Canal de communication pour les messages
- **Partition** : Division d'un topic pour la parallélisation
- **Offset** : Position d'un message dans une partition
- **Consumer Group** : Groupe de consommateurs qui se partagent les messages

### Partitionnement

Les messages avec la même clé vont dans la même partition, garantissant l'ordre de traitement pour ces messages.

## 🧪 Expérimentations suggérées

### 1. Multiple Consommateurs (Load Balancing)

Lancez plusieurs instances du consommateur dans des terminaux différents :

```bash
# Terminal 1
python3 consommateur.py

# Terminal 2
python3 consommateur.py

# Terminal 3
python3 consommateur.py
```

Avec le même `GROUP_ID`, Kafka distribuera automatiquement les messages entre les consommateurs.

### 2. Différents groupes (Broadcasting)

Modifiez le `GROUP_ID` dans différentes instances. Chaque groupe recevra **tous** les messages.

### 3. Persistance

Arrêtez le consommateur, envoyez des messages avec le producteur, puis redémarrez le consommateur. Les messages seront toujours traités grâce à la persistance de Kafka.

## 📊 Architecture

```
┌─────────────┐         ┌─────────────────┐         ┌──────────────┐
│             │         │                 │         │              │
│ Producteur  │ ────▶   │  Kafka Broker   │  ────▶  │ Consommateur │
│ (Publish)   │         │  (Topic/Queue)  │         │ (Subscribe)  │
│             │         │                 │         │              │
└─────────────┘         └─────────────────┘         └──────────────┘
```

### Flux de données

1. Le **producteur** publie des messages vers un **topic** Kafka
2. Kafka stocke les messages dans des **partitions**
3. Les **consommateurs** s'abonnent au topic et consomment les messages
4. Kafka gère automatiquement le **load balancing** entre les consommateurs d'un même groupe

## 🛠️ Fonctionnalités avancées

### Producteur

- ✅ Sérialisation JSON automatique
- ✅ Gestion des erreurs et retry
- ✅ Confirmation de livraison (acks='all')
- ✅ Clés pour le partitionnement
- ✅ Logging détaillé

### Consommateur

- ✅ Désérialisation JSON automatique
- ✅ Commit automatique des offsets
- ✅ Lecture depuis le début (earliest)
- ✅ Gestion gracieuse de l'arrêt (Ctrl+C)
- ✅ Support pour consumer groups
- ✅ Logging détaillé

## 📚 Pour aller plus loin

### Améliorations possibles

1. **Gestion d'erreurs avancée** : Dead letter queue pour les messages en échec
2. **Schémas** : Utiliser Apache Avro pour valider la structure des messages
3. **Monitoring** : Intégrer Prometheus pour surveiller les métriques
4. **Sécurité** : Ajouter l'authentification SSL/SASL
5. **Transactions** : Garantir l'exactitude des traitements (exactly-once semantics)

### Ressources

- [Documentation Apache Kafka](https://kafka.apache.org/documentation/)
- [kafka-python Documentation](https://kafka-python.readthedocs.io/)
- [Confluent Kafka Tutorials](https://kafka-tutorials.confluent.io/)

## 📝 Licence

Ce projet est à usage éducatif uniquement.

## 🤝 Contribution

N'hésitez pas à expérimenter et modifier le code pour mieux comprendre Kafka !

---

**Bon apprentissage ! 🎓**

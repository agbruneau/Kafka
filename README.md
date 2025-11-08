# Pub-Sub avec Kafka et Python

Ce projet est un exemple éducatif d'implémentation du pattern **Pub-Sub (Publisher-Subscriber)** en utilisant **Apache Kafka** et **Python**.

## 📋 Prérequis

- Python 3.7 ou supérieur
- Apache Kafka installé et en cours d'exécution (localement ou à distance)

### Installation de Kafka (optionnel)

Si vous n'avez pas Kafka installé, vous pouvez utiliser Docker :

```bash
# Démarrer Kafka avec Docker Compose
docker-compose up -d
```

Ou installer Kafka manuellement depuis [kafka.apache.org](https://kafka.apache.org/downloads)

## 🚀 Installation

1. Installer les dépendances Python :

```bash
pip install -r requirements.txt
```

## 📖 Utilisation

### 1. Démarrer le consommateur

Dans un premier terminal, démarrez le consommateur qui écoutera les messages :

```bash
python consommateur.py
```

### 2. Envoyer des messages avec le producteur

Dans un second terminal, lancez le producteur pour envoyer des messages :

```bash
python producteur.py
```

Vous devriez voir les messages apparaître dans le terminal du consommateur !

## 🔧 Configuration

Vous pouvez modifier les paramètres par défaut dans les fichiers :

- **Bootstrap servers** : Adresse du serveur Kafka (par défaut: `localhost:9092`)
- **Topic** : Nom du topic Kafka (par défaut: `mon-topic`)
- **Group ID** : ID du groupe de consommateurs (par défaut: `mon-groupe-consommateur`)

### Exemple de personnalisation

```python
# Dans producteur.py ou consommateur.py
BOOTSTRAP_SERVERS = 'kafka.example.com:9092'
TOPIC = 'mon-topic-personnalise'
```

## 📚 Concepts expliqués

### Producteur (Publisher)
- **Rôle** : Envoie des messages vers un topic Kafka
- **Fonctionnalités** :
  - Sérialisation JSON des messages
  - Gestion des clés pour le partitionnement
  - Retry automatique en cas d'échec
  - Confirmation de réception (acks='all')

### Consommateur (Subscriber)
- **Rôle** : Lit les messages depuis un topic Kafka
- **Fonctionnalités** :
  - Désérialisation JSON des messages
  - Gestion des groupes de consommateurs
  - Commit automatique des offsets
  - Consommation en continu ou limitée

## 🎯 Pattern Pub-Sub

Le pattern **Pub-Sub** permet :
- **Découplage** : Les producteurs et consommateurs ne se connaissent pas directement
- **Scalabilité** : Plusieurs consommateurs peuvent lire les mêmes messages
- **Fiabilité** : Les messages sont persistés et peuvent être relus
- **Distribution** : Les messages peuvent être distribués sur plusieurs partitions

## 📝 Structure du projet

```
.
├── producteur.py      # Script du producteur Kafka
├── consommateur.py    # Script du consommateur Kafka
├── requirements.txt   # Dépendances Python
└── README.md         # Documentation
```

## 🐛 Dépannage

### Erreur de connexion à Kafka

Vérifiez que Kafka est bien démarré :

```bash
# Vérifier si Kafka écoute sur le port 9092
netstat -an | grep 9092
```

### Topic n'existe pas

Kafka créera automatiquement le topic si l'option `auto.create.topics.enable=true` est activée (par défaut).

Sinon, créez-le manuellement :

```bash
kafka-topics.sh --create --topic mon-topic --bootstrap-server localhost:9092
```

## 📖 Ressources

- [Documentation Kafka](https://kafka.apache.org/documentation/)
- [kafka-python Documentation](https://kafka-python.readthedocs.io/)
